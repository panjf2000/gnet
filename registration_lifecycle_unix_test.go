//go:build linux

package gnet

import (
	"errors"
	"io"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/netpoll"
	"github.com/panjf2000/gnet/v2/pkg/queue"
)

func pr766LifecycleWait(t *testing.T, ch <-chan error) error {
	t.Helper()
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	select {
	case err := <-ch:
		return err
	case <-timer.C:
		t.Fatal("operation did not finish")
		return nil
	}
}

func pr766LifecyclePolling(p *netpoll.Poller) error {
	if optimized, ok := any(p).(interface{ Polling() error }); ok {
		return optimized.Polling()
	}
	return any(p).(interface {
		Polling(netpoll.PollEventHandler) error
	}).Polling(
		func(int, netpoll.IOEvent, netpoll.IOFlags) error { return nil })
}

type pr766LifecycleHandler struct {
	BuiltinEventEngine
	opens  atomic.Int32
	closes atomic.Int32
}

func (h *pr766LifecycleHandler) OnOpen(Conn) ([]byte, Action) {
	h.opens.Add(1)
	return []byte("registration-live"), None
}

func (h *pr766LifecycleHandler) OnClose(Conn, error) Action {
	h.closes.Add(1)
	return None
}

func pr766LifecycleLoop(t *testing.T) (*eventloop, *pr766LifecycleHandler) {
	t.Helper()
	p, err := netpoll.OpenPoller()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Error(err)
		}
	})
	h := &pr766LifecycleHandler{}
	el := &eventloop{poller: p, engine: &engine{opts: &Options{}}, eventHandler: h}
	el.connections.init()
	return el, h
}

func pr766LifecycleDup(t *testing.T, raw *net.TCPConn, minimum int) int {
	t.Helper()
	rc, err := raw.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	var fd int
	var dupErr error
	if err := rc.Control(func(original uintptr) {
		fd, dupErr = unix.FcntlInt(original, unix.F_DUPFD_CLOEXEC, minimum)
	}); err != nil {
		t.Fatal(err)
	}
	if dupErr != nil {
		t.Fatal(dupErr)
	}
	return fd
}

func pr766LifecycleAssertClosed(t *testing.T, fd int) {
	t.Helper()
	if _, err := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0); !errors.Is(err, unix.EBADF) {
		t.Fatalf("descriptor %d: F_GETFD = %v, want EBADF", fd, err)
	}
}

func pr766LifecycleReuse(t *testing.T, raw *net.TCPConn, fd int, action func()) {
	t.Helper()
	witness := pr766LifecycleDup(t, raw, fd)
	if witness != fd {
		if err := unix.Close(witness); err != nil {
			t.Error(err)
		}
		t.Fatalf("reuse precondition: allocated %d, want %d", witness, fd)
	}
	// Only close a still-live witness: a failing candidate may have closed it.
	defer func() {
		if _, err := unix.FcntlInt(uintptr(witness), unix.F_GETFD, 0); err == nil {
			if err := unix.Close(witness); err != nil {
				t.Error(err)
			}
		} else if !errors.Is(err, unix.EBADF) {
			t.Error(err)
		}
	}()
	if _, err := unix.FcntlInt(uintptr(witness), unix.F_GETFD, 0); err != nil {
		t.Fatalf("live reuse witness: %v", err)
	}
	action()
	if _, err := unix.FcntlInt(uintptr(witness), unix.F_GETFD, 0); err != nil {
		t.Fatalf("terminal request destroyed reused descriptor: %v", err)
	}
}

func TestPR766LifecycleRetiredRequestDiscardsDupOnce(t *testing.T) {
	el, h := pr766LifecycleLoop(t)
	raw := pr766v2TCPPair(t)
	fd := pr766LifecycleDup(t, raw, 0)
	c := &conn{fd: fd, loop: el, pollAttachment: netpoll.PollAttachment{FD: fd}}
	r := &connRegistration{c: c, done: make(chan struct{})}
	r.publication.Lock()
	r.submitErr = el.poller.Trigger(queue.HighPriority, el.register, r)
	r.publication.Unlock()
	if r.submitErr != nil {
		r.discard(r.submitErr)
		t.Fatal(r.submitErr)
	}
	if err := el.poller.Close(); err != nil {
		t.Fatal(err)
	}
	if awaitPollerRegister(r.done, el.poller.Done()) {
		t.Fatal("dropped registration reported completion before its finalizer")
	}
	// Exercise the finalizer at the documented Done boundary, rather than
	// pretending a public scheduling seam exists for registerConn's local r.
	r.discard(errorx.ErrEngineShutdown)
	<-r.done
	if !errors.Is(r.err, errorx.ErrEngineShutdown) {
		t.Fatalf("discard result = %v, want ErrEngineShutdown", r.err)
	}
	pr766LifecycleAssertClosed(t, fd)
	pr766LifecycleReuse(t, raw, fd, func() {
		r.discard(unix.EIO)
		r.complete(nil)
	})
	if !errors.Is(r.err, errorx.ErrEngineShutdown) || h.opens.Load() != 0 || h.closes.Load() != 0 {
		t.Fatalf("discard changed result or called handlers: err=%v opens=%d closes=%d", r.err, h.opens.Load(), h.closes.Load())
	}
}

func TestPR766LifecycleWakeErrorStaleCallbackCannotTouchReuse(t *testing.T) {
	el, h := pr766LifecycleLoop(t)
	raw := pr766v2TCPPair(t)
	fd := pr766LifecycleDup(t, raw, 0)
	c := &conn{fd: fd, loop: el, pollAttachment: netpoll.PollAttachment{FD: fd}}
	// Obtain a real kernel failure, not an error label invented by the test.
	_, writeErr := unix.Write(-1, []byte{1})
	if !errors.Is(writeErr, unix.EBADF) {
		t.Fatalf("negative syscall control = %v, want EBADF", writeErr)
	}
	submitErr := os.NewSyscallError("write", writeErr)
	r := &connRegistration{c: c, done: make(chan struct{})}
	r.publication.Lock()
	r.submitErr = submitErr
	r.publication.Unlock()
	r.discard(submitErr)
	pr766LifecycleAssertClosed(t, fd)
	pr766LifecycleReuse(t, raw, fd, func() {
		// This is the actual stale callback left queued after a wake failure.
		if err := el.register(r); err != nil {
			t.Fatalf("stale callback = %v, want nil (error already published)", err)
		}
		r.discard(unix.EIO)
	})
	if !errors.Is(r.err, unix.EBADF) || h.opens.Load() != 0 || h.closes.Load() != 0 || el.countConn() != 0 {
		t.Fatalf("wake error became registration: err=%v opens=%d closes=%d connections=%d", r.err, h.opens.Load(), h.closes.Load(), el.countConn())
	}
}

func TestPR766LifecycleRegisterRejectedAfterRetireClosesDup(t *testing.T) {
	el, h := pr766LifecycleLoop(t)
	raw := pr766v2TCPPair(t)
	fd := pr766LifecycleDup(t, raw, 0)
	if err := el.poller.Close(); err != nil {
		t.Fatal(err)
	}
	c := &conn{fd: fd, loop: el, pollAttachment: netpoll.PollAttachment{FD: fd}}
	if err := el.registerConn(c, queue.HighPriority); !errors.Is(err, errorx.ErrEngineShutdown) {
		t.Fatalf("registerConn after retirement = %v, want ErrEngineShutdown", err)
	}
	pr766LifecycleAssertClosed(t, fd)
	pr766LifecycleReuse(t, raw, fd, func() {
		if err := el.poller.Close(); err != nil {
			t.Fatal(err)
		}
	})
	if h.opens.Load() != 0 || el.countConn() != 0 {
		t.Fatal("rejected registration reached OnOpen or connection table")
	}
}

func TestPR766LifecycleSubmissionPublicationPrecedesCallback(t *testing.T) {
	el, h := pr766LifecycleLoop(t)
	raw := pr766v2TCPPair(t)
	fd := pr766LifecycleDup(t, raw, 0)
	c := &conn{fd: fd, loop: el, pollAttachment: netpoll.PollAttachment{FD: fd}}
	r := &connRegistration{c: c, done: make(chan struct{})}
	r.publication.Lock()
	started := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		close(started)
		finished <- el.register(r)
	}()
	<-started
	_, writeErr := unix.Write(-1, []byte{1})
	if !errors.Is(writeErr, unix.EBADF) {
		r.publication.Unlock()
		t.Fatalf("negative write control = %v, want EBADF", writeErr)
	}
	r.submitErr = os.NewSyscallError("write", writeErr)
	r.publication.Unlock()
	if err := pr766LifecycleWait(t, finished); err != nil {
		t.Fatalf("failed-submission callback returned %v", err)
	}
	select {
	case <-r.done:
	default:
		t.Fatal("callback did not publish terminal submission failure")
	}
	if !errors.Is(r.err, unix.EBADF) || h.opens.Load() != 0 || el.countConn() != 0 {
		t.Fatalf("callback ignored submission outcome: err=%v opens=%d connections=%d", r.err, h.opens.Load(), el.countConn())
	}
	pr766LifecycleAssertClosed(t, fd)
}

func TestPR766LifecycleAddReadFailureIsRegistrationError(t *testing.T) {
	el, h := pr766LifecycleLoop(t)
	// Regular files are valid descriptors but epoll refuses them with EPERM.
	f, err := os.CreateTemp(t.TempDir(), "not-pollable")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	})
	fd, err := unix.FcntlInt(f.Fd(), unix.F_DUPFD_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := el.poller.AddRead(&netpoll.PollAttachment{FD: fd}, false); !errors.Is(err, unix.EPERM) {
		if closeErr := unix.Close(fd); closeErr != nil {
			t.Error(closeErr)
		}
		t.Fatalf("negative AddRead control = %v, want EPERM", err)
	}
	joined := make(chan error, 1)
	go func() { joined <- pr766LifecyclePolling(el.poller) }()
	t.Cleanup(func() {
		if err := el.poller.Trigger(queue.HighPriority, func(any) error { return errorx.ErrEngineShutdown }, nil); err != nil {
			t.Error(err)
		}
		if err := pr766LifecycleWait(t, joined); !errors.Is(err, errorx.ErrEngineShutdown) {
			t.Errorf("Polling = %v, want ErrEngineShutdown", err)
		}
	})
	c := &conn{fd: fd, loop: el, pollAttachment: netpoll.PollAttachment{FD: fd}}
	result := make(chan error, 1)
	go func() { result <- el.registerConn(c, queue.HighPriority) }()
	if err := pr766LifecycleWait(t, result); !errors.Is(err, unix.EPERM) {
		t.Fatalf("registration = %v, want AddRead EPERM, never false success", err)
	}
	pr766LifecycleAssertClosed(t, fd)
	if h.opens.Load() != 0 || el.countConn() != 0 {
		t.Fatal("failed AddRead reached OnOpen or connection table")
	}
}

func TestPR766LifecycleClientEnrollSuccessAndStopCloses(t *testing.T) {
	ln, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := ln.Close(); err != nil {
			t.Error(err)
		}
	})
	raw, err := net.DialTCP("tcp", nil, ln.Addr().(*net.TCPAddr))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := raw.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Error(err)
		}
	})
	peer, err := ln.AcceptTCP()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := peer.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := peer.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	rc, err := raw.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	h := &pr766LifecycleHandler{}
	cli, err := NewClient(h, WithNumEventLoop(1))
	if err != nil {
		t.Fatal(err)
	}
	if err := cli.Start(); err != nil {
		t.Fatal(err)
	}
	stopped := false
	t.Cleanup(func() {
		if !stopped {
			if err := cli.Stop(); err != nil {
				t.Error(err)
			}
		}
	})
	registered, err := cli.Enroll(raw)
	if err != nil || registered == nil {
		t.Fatalf("Enroll = (%v, %v), want connection and nil", registered, err)
	}
	fd := registered.Fd()
	if _, err := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0); err != nil {
		t.Fatal(err)
	}
	if err := rc.Control(func(uintptr) {}); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("original after Enroll = %v, want net.ErrClosed", err)
	}
	greeting := make([]byte, len("registration-live"))
	if _, err := io.ReadFull(peer, greeting); err != nil {
		t.Fatal(err)
	}
	if string(greeting) != "registration-live" {
		t.Fatalf("greeting = %q", greeting)
	}
	written := make(chan error, 1)
	if err := registered.AsyncWrite([]byte("retained-dup"), func(_ Conn, err error) error { written <- err; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := pr766LifecycleWait(t, written); err != nil {
		t.Fatal(err)
	}
	payload := make([]byte, len("retained-dup"))
	if _, err := io.ReadFull(peer, payload); err != nil {
		t.Fatal(err)
	}
	if string(payload) != "retained-dup" {
		t.Fatalf("payload = %q", payload)
	}
	err = cli.Stop()
	stopped = true
	if err != nil {
		t.Fatalf("Stop = %v", err)
	}
	pr766LifecycleAssertClosed(t, fd)
	var buf [1]byte
	if n, err := peer.Read(buf[:]); n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("peer after Stop = (%d, %v), want (0, EOF)", n, err)
	}
	if h.opens.Load() != 1 || h.closes.Load() != 1 {
		t.Fatalf("callbacks: opens=%d closes=%d, want 1 each", h.opens.Load(), h.closes.Load())
	}
	if err := registered.Close(); !errors.Is(err, errorx.ErrEngineShutdown) {
		t.Fatalf("Close after Stop = %v, want ErrEngineShutdown", err)
	}
}
