// Copyright (c) 2026 Telego contributors.
// Licensed under the Apache License, Version 2.0.

//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package gnet

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"syscall"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/queue"
	"github.com/panjf2000/gnet/v2/pkg/socket"
	"golang.org/x/sys/unix"
)

// A pending registration exclusively owns its unregistered descriptor. Claim
// and cancellation linearize under the loop's registration mutex. A stale
// queued request has a nil connection and cannot close a reused descriptor.
type registration struct {
	loop   *eventloop
	conn   *conn
	done   chan struct{}
	result RegisteredResult
	once   sync.Once
}

func (r *registration) complete(result RegisteredResult) {
	r.once.Do(func() { r.result = result; close(r.done) })
}

func disposeUnregistered(c *conn) {
	_ = unix.Close(c.fd)
	c.release()
}

func (el *eventloop) submitRegistration(c *conn, priority queue.EventPriority) (*registration, error) {
	r := &registration{loop: el, conn: c, done: make(chan struct{})}
	el.registrationMu.Lock()
	if el.registrationsClosed || el.engine.isStopping() {
		r.conn = nil
		el.registrationMu.Unlock()
		disposeUnregistered(c)
		r.complete(RegisteredResult{Err: errorx.ErrEngineInShutdown})
		return r, errorx.ErrEngineInShutdown
	}
	if el.registrations == nil {
		el.registrations = make(map[*registration]struct{})
	}
	el.registrations[r] = struct{}{}
	el.registrationMu.Unlock()
	err := el.poller.Trigger(priority, el.register, r)
	if err != nil {
		r.cancel(err)
	}
	return r, err
}

func (r *registration) cancel(err error) {
	el := r.loop
	el.registrationMu.Lock()
	c := r.conn
	if c != nil {
		r.conn = nil
		delete(el.registrations, r)
	}
	el.registrationMu.Unlock()
	if c != nil {
		disposeUnregistered(c)
		r.complete(RegisteredResult{Err: err})
	}
}

func (r *registration) run() error {
	el := r.loop
	el.registrationMu.Lock()
	c := r.conn
	r.conn = nil
	delete(el.registrations, r)
	el.registrationMu.Unlock()
	if c == nil {
		return nil
	}
	err := el.register0(c)
	if err != nil {
		if el.connections.getConn(c.fd) == c {
			_ = el.close(c, err)
		}
		r.complete(RegisteredResult{Err: err})
		return err
	}
	if el.connections.getConn(c.fd) != c {
		r.complete(RegisteredResult{Err: net.ErrClosed})
		return nil
	}
	r.complete(RegisteredResult{Conn: c})
	return nil
}

func (el *eventloop) retireRegistrations() {
	el.registrationMu.Lock()
	el.registrationsClosed = true
	pending := make(map[*registration]*conn, len(el.registrations))
	for r := range el.registrations {
		pending[r] = r.conn
		r.conn = nil
	}
	el.registrations = nil
	el.registrationMu.Unlock()
	for r, c := range pending {
		disposeUnregistered(c)
		r.complete(RegisteredResult{Err: errorx.ErrEngineInShutdown})
	}
}

func (eng *engine) beginEnrollment() bool {
	eng.admissionMu.Lock()
	defer eng.admissionMu.Unlock()
	if eng.isStopping() {
		return false
	}
	eng.enrollments.Add(1)
	return true
}

// Go 1.20 has no context.AfterFunc. Only cancelable caller contexts need a
// short-lived merger; it is joined before this enrollment releases ownership.
func enrollmentContext(caller, lifetime context.Context) (context.Context, func()) {
	if caller == nil || caller.Done() == nil {
		return lifetime, func() {}
	}
	ctx, cancel := context.WithCancelCause(lifetime)
	if caller.Err() != nil {
		cancel(context.Cause(caller))
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case <-caller.Done():
			cancel(context.Cause(caller))
		case <-ctx.Done():
		}
	}()
	return ctx, func() { cancel(context.Canceled); <-done }
}

func (el *eventloop) enrollConnection(caller context.Context, c net.Conn, addr net.Addr, metadata any, priority queue.EventPriority, tune bool) RegisteredResult {
	ctx, cancel := enrollmentContext(caller, el.engine.concurrency.ctx)
	defer cancel()
	if c != nil {
		defer func() { _ = c.Close() }()
	}
	if err := ctx.Err(); err != nil {
		return RegisteredResult{Err: context.Cause(ctx)}
	}
	if c == nil {
		var err error
		c, err = (&net.Dialer{}).DialContext(ctx, addr.Network(), addr.String())
		if err != nil {
			if ctx.Err() != nil {
				err = context.Cause(ctx)
			}
			return RegisteredResult{Err: err}
		}
		defer func() { _ = c.Close() }()
	}
	gc, err := el.duplicateConnection(c, metadata, tune)
	if err != nil {
		return RegisteredResult{Err: err}
	}
	if err = ctx.Err(); err != nil {
		disposeUnregistered(gc)
		return RegisteredResult{Err: context.Cause(ctx)}
	}
	r, err := el.submitRegistration(gc, priority)
	if err == nil {
		select {
		case <-r.done:
		case <-ctx.Done():
			r.cancel(context.Cause(ctx))
		}
	}
	<-r.done
	return r.result
}

func (el *eventloop) duplicateConnection(c net.Conn, metadata any, tune bool) (gc *conn, err error) {
	sc, ok := c.(syscall.Conn)
	if !ok {
		return nil, errors.New("failed to convert net.Conn to syscall.Conn")
	}
	raw, err := sc.SyscallConn()
	if err != nil {
		return nil, err
	}
	fd := -1
	defer func() {
		if fd >= 0 {
			_ = unix.Close(fd)
		}
	}()
	controlErr := raw.Control(func(original uintptr) { fd, err = socket.Dup(int(original)) })
	if err != nil {
		return nil, err
	}
	if controlErr != nil {
		return nil, controlErr
	}
	if tune {
		if el.engine.opts.SocketSendBuffer > 0 {
			if err = socket.SetSendBuffer(fd, el.engine.opts.SocketSendBuffer); err != nil {
				return nil, err
			}
		}
		if el.engine.opts.SocketRecvBuffer > 0 {
			if err = socket.SetRecvBuffer(fd, el.engine.opts.SocketRecvBuffer); err != nil {
				return nil, err
			}
		}
	}
	var sa unix.Sockaddr
	switch c.(type) {
	case *net.UnixConn:
		sa, _, _, err = socket.GetUnixSockAddr(c.RemoteAddr().Network(), c.RemoteAddr().String())
		if err != nil {
			return nil, err
		}
		local := *c.LocalAddr().(*net.UnixAddr)
		local.Name = c.RemoteAddr().String() + "." + strconv.Itoa(fd)
		gc = newStreamConn("unix", fd, el, sa, &local, c.RemoteAddr())
	case *net.TCPConn:
		if tune && el.engine.opts.TCPNoDelay == TCPNoDelay {
			if err = socket.SetNoDelay(fd, 1); err != nil {
				return nil, err
			}
		}
		if tune && el.engine.opts.TCPKeepAlive > 0 {
			o := el.engine.opts
			if err = setKeepAlive(fd, true, o.TCPKeepAlive, o.TCPKeepInterval, o.TCPKeepCount); err != nil {
				return nil, err
			}
		}
		sa, _, _, _, err = socket.GetTCPSockAddr(c.RemoteAddr().Network(), c.RemoteAddr().String())
		if err != nil {
			return nil, err
		}
		gc = newStreamConn("tcp", fd, el, sa, c.LocalAddr(), c.RemoteAddr())
	case *net.UDPConn:
		sa, _, _, _, err = socket.GetUDPSockAddr(c.RemoteAddr().Network(), c.RemoteAddr().String())
		if err != nil {
			return nil, err
		}
		gc = newUDPConn(fd, el, c.LocalAddr(), sa, true)
	default:
		return nil, fmt.Errorf("%w: %T", errorx.ErrUnsupportedProtocol, c)
	}
	fd = -1
	gc.SetContext(metadata)
	gc.SetSafeContext(metadata)
	return gc, nil
}
