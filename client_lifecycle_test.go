// Copyright (c) 2026 Telego contributors.
// Licensed under the Apache License, Version 2.0. See LICENSE.

//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package gnet

import (
	"context"
	"errors"
	"io"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/queue"
	"golang.org/x/sys/unix"
)

type lifecycleEvents struct {
	BuiltinEventEngine
	onBoot    func(Engine) Action
	onOpen    func(Conn) ([]byte, Action)
	shutdowns atomic.Int32
}

func (h *lifecycleEvents) OnBoot(e Engine) Action {
	if h.onBoot != nil {
		return h.onBoot(e)
	}
	return None
}
func (h *lifecycleEvents) OnOpen(c Conn) ([]byte, Action) {
	if h.onOpen != nil {
		return h.onOpen(c)
	}
	return nil, None
}
func (h *lifecycleEvents) OnShutdown(Engine) { h.shutdowns.Add(1) }

func lifecyclePair(t *testing.T) (net.Conn, net.Conn) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	c, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	p, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close() })
	return c, p
}
func lifecycleWait(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("lifecycle did not complete")
	}
}
func lifecycleClient(t *testing.T, events *lifecycleEvents) *Client {
	t.Helper()
	c, err := NewClient(events, WithNumEventLoop(1))
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Stop() })
	return c
}
func lifecyclePeerEOF(t *testing.T, peer net.Conn) {
	t.Helper()
	_ = peer.SetReadDeadline(time.Now().Add(time.Second))
	var b [1]byte
	if n, err := peer.Read(b[:]); n != 0 || err != io.EOF {
		t.Fatalf("peer read = %d, %v; want EOF", n, err)
	}
}

func TestClientConcurrentStopAndAutomaticDone(t *testing.T) {
	h := &lifecycleEvents{}
	c := lifecycleClient(t, h)
	source, peer := lifecyclePair(t)
	gc, err := c.Enroll(source)
	if err != nil {
		t.Fatal(err)
	}
	if err = gc.EventLoop().Execute(context.Background(), RunnableFunc(func(context.Context) error { return errorx.ErrEngineShutdown })); err != nil {
		t.Fatal(err)
	}
	lifecycleWait(t, c.Done())
	lifecyclePeerEOF(t, peer)
	var wait sync.WaitGroup
	for i := 0; i < 16; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if stopErr := c.Stop(); stopErr != c.Err() {
				t.Errorf("Stop = %v, Err = %v", stopErr, c.Err())
			}
		}()
	}
	wait.Wait()
	if got := h.shutdowns.Load(); got != 1 {
		t.Fatalf("OnShutdown called %d times", got)
	}
	if err = c.Start(); !errors.Is(err, errorx.ErrEngineInShutdown) {
		t.Fatalf("restart = %v", err)
	}
}

func TestClientStopBeforeAndDuringStart(t *testing.T) {
	t.Run("before", func(t *testing.T) {
		c, err := NewClient(&lifecycleEvents{})
		if err != nil {
			t.Fatal(err)
		}
		if err = c.Stop(); err != nil {
			t.Fatal(err)
		}
		lifecycleWait(t, c.Done())
		if err = c.Start(); !errors.Is(err, errorx.ErrEngineInShutdown) {
			t.Fatalf("Start = %v", err)
		}
		source, peer := lifecyclePair(t)
		if _, err = c.Enroll(source); !errors.Is(err, errorx.ErrEngineInShutdown) {
			t.Fatalf("Enroll = %v", err)
		}
		lifecyclePeerEOF(t, peer)
	})
	t.Run("during", func(t *testing.T) {
		entered, release := make(chan struct{}), make(chan struct{})
		h := &lifecycleEvents{onBoot: func(Engine) Action { close(entered); <-release; return None }}
		c, err := NewClient(h, WithNumEventLoop(2))
		if err != nil {
			t.Fatal(err)
		}
		started := make(chan error, 1)
		go func() { started <- c.Start() }()
		lifecycleWait(t, entered)
		stopped := make(chan error, 1)
		go func() { stopped <- c.Stop() }()
		<-c.eng.concurrency.ctx.Done()
		close(release)
		if err = <-started; err != nil {
			t.Fatal(err)
		}
		lifecycleWait(t, c.Done())
		if err = <-stopped; err != nil {
			t.Fatal(err)
		}
		if h.shutdowns.Load() != 1 {
			t.Fatal("shutdown not exactly once")
		}
	})
}

func TestEnrollmentOnOpenActions(t *testing.T) {
	for _, action := range []Action{None, Close, Shutdown} {
		t.Run(map[Action]string{None: "none", Close: "close", Shutdown: "shutdown"}[action], func(t *testing.T) {
			h := &lifecycleEvents{onOpen: func(Conn) ([]byte, Action) { return nil, action }}
			c := lifecycleClient(t, h)
			source, peer := lifecyclePair(t)
			gc, err := c.Enroll(source)
			if action == None {
				if err != nil || gc == nil {
					t.Fatalf("Enroll = %v, %v", gc, err)
				}
				return
			}
			if err == nil || gc != nil {
				t.Fatalf("closed enrollment = %v, %v", gc, err)
			}
			if action == Shutdown {
				lifecycleWait(t, c.Done())
			}
			lifecyclePeerEOF(t, peer)
		})
	}
}

func TestEnrollmentPreCanceledContext(t *testing.T) {
	var opens atomic.Int32
	c := lifecycleClient(t, &lifecycleEvents{onOpen: func(Conn) ([]byte, Action) { opens.Add(1); return nil, None }})
	source, peer := lifecyclePair(t)
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("caller canceled enrollment")
	cancel(cause)
	result, err := c.eng.eventLoops.next(nil).Enroll(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-result:
		if !errors.Is(got.Err, cause) || got.Conn != nil {
			t.Fatalf("Enroll = %#v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled enrollment did not finish")
	}
	if opens.Load() != 0 {
		t.Fatal("pre-canceled context opened a connection")
	}
	lifecyclePeerEOF(t, peer)
}

func TestRegistrationReportsPollerError(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("epoll duplicate-add failure is Linux-specific")
	}
	c := lifecycleClient(t, &lifecycleEvents{})
	source, peer := lifecyclePair(t)
	el := c.eng.eventLoops.next(nil)
	gc, err := el.duplicateConnection(source, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	var result RegisteredResult
	if err = el.Execute(context.Background(), RunnableFunc(func(context.Context) error {
		defer close(done)
		if addErr := el.poller.AddRead(&gc.pollAttachment, false); addErr != nil {
			result.Err = addErr
			return nil
		}
		r := &registration{loop: el, conn: gc, done: make(chan struct{})}
		_ = r.run()
		result = r.result
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	lifecycleWait(t, done)
	if result.Conn != nil || !errors.Is(result.Err, unix.EEXIST) {
		t.Fatalf("duplicate registration = %#v", result)
	}
	_ = source.Close()
	lifecyclePeerEOF(t, peer)
}

type lifecycleUnsupportedConn struct{ net.Conn }

func (c lifecycleUnsupportedConn) SyscallConn() (syscall.RawConn, error) {
	return c.Conn.(syscall.Conn).SyscallConn()
}

type lifecycleControlErrorConn struct {
	net.Conn
	raw     syscall.RawConn
	failure error
}

func (c lifecycleControlErrorConn) SyscallConn() (syscall.RawConn, error) {
	return lifecycleControlError{c.raw, c.failure}, nil
}

type lifecycleControlError struct {
	syscall.RawConn
	failure error
}

func (c lifecycleControlError) Control(f func(uintptr)) error {
	if err := c.RawConn.Control(f); err != nil {
		return err
	}
	return c.failure
}

func TestPostDuplicationFailuresDisposeDescriptor(t *testing.T) {
	c := lifecycleClient(t, &lifecycleEvents{})
	el := c.eng.eventLoops.next(nil)
	for _, controlFailure := range []bool{false, true} {
		source, peer := lifecyclePair(t)
		var wrapped net.Conn = lifecycleUnsupportedConn{source}
		if controlFailure {
			raw, err := source.(syscall.Conn).SyscallConn()
			if err != nil {
				t.Fatal(err)
			}
			wrapped = lifecycleControlErrorConn{source, raw, errors.New("control failed after callback")}
		}
		if gc, err := el.duplicateConnection(wrapped, nil, true); err == nil || gc != nil {
			t.Fatalf("duplicate = %v,%v", gc, err)
		}
		_ = source.Close()
		lifecyclePeerEOF(t, peer)
	}
}

func TestClientDialCanceledByStop(t *testing.T) {
	originalResolver := net.DefaultResolver
	entered := make(chan struct{})
	var once sync.Once
	net.DefaultResolver = &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
		once.Do(func() { close(entered) })
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	t.Cleanup(func() { net.DefaultResolver = originalResolver })
	c := lifecycleClient(t, &lifecycleEvents{})
	result := make(chan error, 1)
	go func() { _, err := c.DialContext("tcp", "gnet-lifecycle.invalid:1234", nil); result <- err }()
	lifecycleWait(t, entered)
	if err := c.Stop(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Dial = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Dial did not return its terminal result")
	}
}

func TestRegistrationCancellationReleasesPendingDescriptor(t *testing.T) {
	c := lifecycleClient(t, &lifecycleEvents{})
	el := c.eng.eventLoops.next(nil)
	entered, release := make(chan struct{}), make(chan struct{})
	if err := el.Execute(context.Background(), RunnableFunc(func(context.Context) error { close(entered); <-release; return nil })); err != nil {
		t.Fatal(err)
	}
	lifecycleWait(t, entered)
	source, peer := lifecyclePair(t)
	gc, err := el.duplicateConnection(source, nil, false)
	if err != nil {
		close(release)
		t.Fatal(err)
	}
	r, err := el.submitRegistration(gc, queue.LowPriority)
	if err != nil {
		close(release)
		t.Fatal(err)
	}
	r.cancel(context.Canceled)
	lifecycleWait(t, r.done)
	_ = source.Close()
	lifecyclePeerEOF(t, peer)
	close(release)
	if !errors.Is(r.result.Err, context.Canceled) || r.result.Conn != nil {
		t.Fatalf("result = %#v", r.result)
	}
}

func TestClientExplicitStopRetiresEnrollmentBeforeOwnerJoin(t *testing.T) {
	c := lifecycleClient(t, &lifecycleEvents{})
	el := c.eng.eventLoops.next(nil)
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	if err := el.Execute(context.Background(), RunnableFunc(func(context.Context) error {
		close(entered)
		<-release
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	lifecycleWait(t, entered)
	source, peer := lifecyclePair(t)
	result, err := el.Enroll(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	stopped := make(chan error, 1)
	go func() { stopped <- c.Stop() }()
	select {
	case got := <-result:
		if got.Err == nil || got.Conn != nil {
			t.Fatalf("result = %#v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("Stop did not cancel enrollment setup")
	}
	lifecyclePeerEOF(t, peer)
	select {
	case <-c.Done():
		t.Fatal("Done closed before owner joined")
	default:
	}
	unblock()
	lifecycleWait(t, c.Done())
	if err = <-stopped; err != nil {
		t.Fatal(err)
	}
	// An immediate EventLoop rejection does not take ownership of the source.
	rejected, _ := lifecyclePair(t)
	if _, err = el.Enroll(context.Background(), rejected); !errors.Is(err, errorx.ErrEngineInShutdown) {
		t.Fatalf("late enrollment = %v", err)
	}
	if err = rejected.SetReadDeadline(time.Now()); err != nil {
		t.Fatalf("rejection closed caller-owned socket: %v", err)
	}
}

func TestClientTerminalErrorIsRetained(t *testing.T) {
	c := lifecycleClient(t, &lifecycleEvents{})
	failure := errors.New("client worker failed")
	c.eng.concurrency.Go(func() error { return failure })
	lifecycleWait(t, c.Done())
	if c.Err() != failure || c.Stop() != failure {
		t.Fatalf("Err = %v, Stop = %v", c.Err(), c.Stop())
	}
}
