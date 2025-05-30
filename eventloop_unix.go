// Copyright (c) 2019 The Gnet Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package gnet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	gio "github.com/panjf2000/gnet/v2/pkg/io"
	"github.com/panjf2000/gnet/v2/pkg/logging"
	"github.com/panjf2000/gnet/v2/pkg/netpoll"
	"github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
	"github.com/panjf2000/gnet/v2/pkg/queue"
	"github.com/panjf2000/gnet/v2/pkg/socket"
)

type eventloop struct {
	listeners    map[int]*listener // listeners
	idx          int               // loop index in the engine loops list
	engine       *engine           // engine in loop
	poller       *netpoll.Poller   // epoll or kqueue
	buffer       []byte            // read packet buffer whose capacity is set by user, default value is 64KB
	connections  connMatrix        // loop connections storage
	eventHandler EventHandler      // user eventHandler
}

func (el *eventloop) Register(ctx context.Context, addr net.Addr) (<-chan RegisteredResult, error) {
	if el.engine.isShutdown() {
		return nil, errorx.ErrEngineInShutdown
	}
	if addr == nil {
		return nil, errorx.ErrInvalidNetworkAddress
	}
	return el.enroll(nil, addr, FromContext(ctx))
}

func (el *eventloop) Enroll(ctx context.Context, c net.Conn) (<-chan RegisteredResult, error) {
	if el.engine.isShutdown() {
		return nil, errorx.ErrEngineInShutdown
	}
	if c == nil {
		return nil, errorx.ErrInvalidNetConn
	}
	return el.enroll(c, c.RemoteAddr(), FromContext(ctx))
}

func (el *eventloop) Execute(ctx context.Context, runnable Runnable) error {
	if el.engine.isShutdown() {
		return errorx.ErrEngineInShutdown
	}
	if runnable == nil {
		return errorx.ErrNilRunnable
	}
	return el.poller.Trigger(queue.LowPriority, func(any) error {
		return runnable.Run(ctx)
	}, nil)
}

func (el *eventloop) Schedule(context.Context, Runnable, time.Duration) error {
	return errorx.ErrUnsupportedOp
}

func (el *eventloop) Close(c Conn) error {
	return el.close(c.(*conn), nil)
}

func (el *eventloop) getLogger() logging.Logger {
	return el.engine.opts.Logger
}

func (el *eventloop) countConn() int32 {
	return el.connections.loadCount()
}

func (el *eventloop) closeConns() {
	// Close loops and all outstanding connections
	el.connections.iterate(func(c *conn) bool {
		_ = el.close(c, nil)
		return true
	})
}

type connWithCallback struct {
	c  *conn
	cb func()
}

func (el *eventloop) enroll(c net.Conn, addr net.Addr, ctx any) (resCh chan RegisteredResult, err error) {
	resCh = make(chan RegisteredResult, 1)
	err = goroutine.DefaultWorkerPool.Submit(func() {
		defer close(resCh)

		var err error
		if c == nil {
			if c, err = net.Dial(addr.Network(), addr.String()); err != nil {
				resCh <- RegisteredResult{Err: err}
				return
			}
		}
		defer c.Close() //nolint:errcheck

		sc, ok := c.(syscall.Conn)
		if !ok {
			resCh <- RegisteredResult{
				Err: fmt.Errorf("failed to assert syscall.Conn from net.Conn: %s", addr.String()),
			}
			return
		}
		rc, err := sc.SyscallConn()
		if err != nil {
			resCh <- RegisteredResult{Err: err}
			return
		}

		var dupFD int
		err1 := rc.Control(func(fd uintptr) {
			dupFD, err = unix.Dup(int(fd))
		})
		if err != nil {
			resCh <- RegisteredResult{Err: err}
			return
		}
		if err1 != nil {
			resCh <- RegisteredResult{Err: err1}
			return
		}

		var (
			sockAddr unix.Sockaddr
			gc       *conn
		)
		switch c.(type) {
		case *net.UnixConn:
			sockAddr, _, _, err = socket.GetUnixSockAddr(c.RemoteAddr().Network(), c.RemoteAddr().String())
			if err != nil {
				resCh <- RegisteredResult{Err: err}
				return
			}
			ua := c.LocalAddr().(*net.UnixAddr)
			ua.Name = c.RemoteAddr().String() + "." + strconv.Itoa(dupFD)
			gc = newStreamConn("unix", dupFD, el, sockAddr, c.LocalAddr(), c.RemoteAddr())
		case *net.TCPConn:
			sockAddr, _, _, _, err = socket.GetTCPSockAddr(c.RemoteAddr().Network(), c.RemoteAddr().String())
			if err != nil {
				resCh <- RegisteredResult{Err: err}
				return
			}
			gc = newStreamConn("tcp", dupFD, el, sockAddr, c.LocalAddr(), c.RemoteAddr())
		case *net.UDPConn:
			sockAddr, _, _, _, err = socket.GetUDPSockAddr(c.RemoteAddr().Network(), c.RemoteAddr().String())
			if err != nil {
				resCh <- RegisteredResult{Err: err}
				return
			}
			gc = newUDPConn(dupFD, el, c.LocalAddr(), sockAddr, true)
		default:
			resCh <- RegisteredResult{Err: fmt.Errorf("unknown type of conn: %T", c)}
			return
		}

		gc.ctx = ctx

		connOpened := make(chan struct{})
		ccb := &connWithCallback{c: gc, cb: func() {
			close(connOpened)
		}}
		if err := el.poller.Trigger(queue.LowPriority, el.register, ccb); err != nil {
			gc.Close() //nolint:errcheck
			resCh <- RegisteredResult{Err: err}
			return
		}
		<-connOpened

		resCh <- RegisteredResult{Conn: gc}
	})
	return
}

func (el *eventloop) register(a any) error {
	c, ok := a.(*conn)
	if !ok {
		ccb := a.(*connWithCallback)
		c = ccb.c
		defer ccb.cb()
	}
	return el.register0(c)
}

func (el *eventloop) register0(c *conn) error {
	addEvents := el.poller.AddRead
	if el.engine.opts.EdgeTriggeredIO {
		addEvents = el.poller.AddReadWrite
	}
	if err := addEvents(&c.pollAttachment, el.engine.opts.EdgeTriggeredIO); err != nil {
		_ = unix.Close(c.fd)
		c.release()
		return err
	}
	el.connections.addConn(c, el.idx)
	if c.isDatagram && c.remote != nil {
		return nil
	}
	return el.open(c)
}

func (el *eventloop) open(c *conn) error {
	c.opened = true

	out, action := el.eventHandler.OnOpen(c)
	if out != nil {
		if err := c.open(out); err != nil {
			return err
		}
	}

	if !c.outboundBuffer.IsEmpty() && !el.engine.opts.EdgeTriggeredIO {
		if err := el.poller.ModReadWrite(&c.pollAttachment, false); err != nil {
			return err
		}
	}

	return el.handleAction(c, action)
}

func (el *eventloop) read0(a any) error {
	return el.read(a.(*conn))
}

func (el *eventloop) read(c *conn) error {
	if !c.opened {
		return nil
	}

	var recv int
	isET := el.engine.opts.EdgeTriggeredIO
	chunk := el.engine.opts.EdgeTriggeredIOChunk
loop:
	n, err := unix.Read(c.fd, el.buffer)
	if err != nil || n == 0 {
		if err == unix.EAGAIN {
			return nil
		}
		if n == 0 {
			err = io.EOF
		}
		return el.close(c, os.NewSyscallError("read", err))
	}
	recv += n

	c.buffer = el.buffer[:n]
	action := el.eventHandler.OnTraffic(c)
	switch action {
	case None:
	case Close:
		return el.close(c, nil)
	case Shutdown:
		return errorx.ErrEngineShutdown
	}
	_, _ = c.inboundBuffer.Write(c.buffer)
	c.buffer = c.buffer[:0]

	if c.isEOF || (isET && recv < chunk) {
		goto loop
	}

	// To prevent infinite reading in ET mode and starving other events,
	// we need to set up threshold for the maximum read bytes per connection
	// on each event-loop. If the threshold is reached and there are still
	// unread data in the socket buffer, we must issue another read event manually.
	if isET && n == len(el.buffer) {
		return el.poller.Trigger(queue.LowPriority, el.read0, c)
	}

	return nil
}

func (el *eventloop) write0(a any) error {
	return el.write(a.(*conn))
}

// The default value of UIO_MAXIOV/IOV_MAX is 1024 on Linux and most BSD-like OSs.
const iovMax = 1024

func (el *eventloop) write(c *conn) error {
	if c.outboundBuffer.IsEmpty() {
		return nil
	}

	isET := el.engine.opts.EdgeTriggeredIO
	chunk := el.engine.opts.EdgeTriggeredIOChunk
	var (
		n    int
		sent int
		err  error
	)
loop:
	iov, _ := c.outboundBuffer.Peek(-1)
	if len(iov) > 1 {
		if len(iov) > iovMax {
			iov = iov[:iovMax]
		}
		n, err = gio.Writev(c.fd, iov)
	} else {
		n, err = unix.Write(c.fd, iov[0])
	}
	_, _ = c.outboundBuffer.Discard(n)
	switch err {
	case nil:
	case unix.EAGAIN:
		return nil
	default:
		return el.close(c, os.NewSyscallError("write", err))
	}
	sent += n

	if isET && !c.outboundBuffer.IsEmpty() && sent < chunk {
		goto loop
	}

	// All data have been sent, it's no need to monitor the writable events for LT mode,
	// remove the writable event from poller to help the future event-loops if necessary.
	if !isET && c.outboundBuffer.IsEmpty() {
		return el.poller.ModRead(&c.pollAttachment, false)
	}

	// To prevent infinite writing in ET mode and starving other events,
	// we need to set up threshold for the maximum write bytes per connection
	// on each event-loop. If the threshold is reached and there are still
	// pending data to write, we must issue another write event manually.
	if isET && !c.outboundBuffer.IsEmpty() {
		return el.poller.Trigger(queue.HighPriority, el.write0, c)
	}

	return nil
}

func (el *eventloop) close(c *conn, err error) error {
	if !c.opened || el.connections.getConn(c.fd) == nil {
		return nil // ignore stale connections
	}

	el.connections.delConn(c)
	action := el.eventHandler.OnClose(c, err)

	// Send residual data in buffer back to the remote before actually closing the connection.
	for !c.outboundBuffer.IsEmpty() {
		iov, _ := c.outboundBuffer.Peek(0)
		if len(iov) > iovMax {
			iov = iov[:iovMax]
		}
		n, err := gio.Writev(c.fd, iov)
		if err != nil {
			break
		}
		_, _ = c.outboundBuffer.Discard(n)
	}

	c.release()

	var errStr strings.Builder
	err0, err1 := el.poller.Delete(c.fd), unix.Close(c.fd)
	if err0 != nil {
		err0 = fmt.Errorf("failed to delete fd=%d from poller in event-loop(%d): %v",
			c.fd, el.idx, os.NewSyscallError("delete", err0))
		errStr.WriteString(err0.Error())
		errStr.WriteString(" | ")
	}
	if err1 != nil {
		err1 = fmt.Errorf("failed to close fd=%d in event-loop(%d): %v",
			c.fd, el.idx, os.NewSyscallError("close", err1))
		errStr.WriteString(err1.Error())
	}
	if errStr.Len() > 0 {
		return errors.New(strings.TrimSuffix(errStr.String(), " | "))
	}

	return el.handleAction(c, action)
}

func (el *eventloop) wake(c *conn) error {
	if !c.opened || el.connections.getConn(c.fd) == nil {
		return nil // ignore stale connections
	}

	action := el.eventHandler.OnTraffic(c)

	return el.handleAction(c, action)
}

func (el *eventloop) ticker(ctx context.Context) {
	var (
		action Action
		delay  time.Duration
		timer  *time.Timer
	)
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		delay, action = el.eventHandler.OnTick()
		switch action {
		case None, Close:
		case Shutdown:
			// It seems reasonable to mark this as low-priority, waiting for some tasks like asynchronous writes
			// to finish up before shutting down the service.
			err := el.poller.Trigger(queue.LowPriority, func(_ any) error { return errorx.ErrEngineShutdown }, nil)
			el.getLogger().Debugf("failed to enqueue shutdown signal of high-priority for event-loop(%d): %v", el.idx, err)
		}
		if timer == nil {
			timer = time.NewTimer(delay)
		} else {
			timer.Reset(delay)
		}
		select {
		case <-ctx.Done():
			el.getLogger().Debugf("stopping ticker in event-loop(%d) from Engine, error:%v", el.idx, ctx.Err())
			return
		case <-timer.C:
		}
	}
}

func (el *eventloop) readUDP(fd int, _ netpoll.IOEvent, _ netpoll.IOFlags) error {
	n, sa, err := unix.Recvfrom(fd, el.buffer, 0)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		return fmt.Errorf("failed to read UDP packet from fd=%d in event-loop(%d), %v",
			fd, el.idx, os.NewSyscallError("recvfrom", err))
	}
	var c *conn
	if ln, ok := el.listeners[fd]; ok {
		c = newUDPConn(fd, el, ln.addr, sa, false)
	} else {
		c = el.connections.getConn(fd)
	}
	c.buffer = el.buffer[:n]
	action := el.eventHandler.OnTraffic(c)
	if c.remote != nil {
		c.release()
	}
	if action == Shutdown {
		return errorx.ErrEngineShutdown
	}
	return nil
}

func (el *eventloop) handleAction(c *conn, action Action) error {
	switch action {
	case None:
		return nil
	case Close:
		return el.close(c, nil)
	case Shutdown:
		return errorx.ErrEngineShutdown
	default:
		return nil
	}
}

/*
func (el *eventloop) execCmd(a any) (err error) {
	cmd := a.(*asyncCmd)
	c := el.connections.getConnByGFD(cmd.fd)
	if c == nil || c.gfd != cmd.fd {
		return errorx.ErrInvalidConn
	}

	defer func() {
		if cmd.cb != nil {
			_ = cmd.cb(c, err)
		}
	}()

	switch cmd.typ {
	case asyncCmdClose:
		return el.close(c, nil)
	case asyncCmdWake:
		return el.wake(c)
	case asyncCmdWrite:
		_, err = c.Write(cmd.param.([]byte))
	case asyncCmdWritev:
		_, err = c.Writev(cmd.param.([][]byte))
	default:
		return errorx.ErrUnsupportedOp
	}
	return
}
*/
