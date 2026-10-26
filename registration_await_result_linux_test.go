//go:build linux

package gnet

import (
	"errors"
	"testing"

	"golang.org/x/sys/unix"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/netpoll"
	"github.com/panjf2000/gnet/v2/pkg/queue"
)

func TestPR766AwaitResultRetirementOwnsFinalization(t *testing.T) {
	t.Run("admitted_then_dropped", func(t *testing.T) {
		el, h := pr766LifecycleLoop(t)
		raw := pr766v2TCPPair(t)
		fd := pr766LifecycleDup(t, raw, 0)
		c := &conn{fd: fd, loop: el, pollAttachment: netpoll.PollAttachment{FD: fd}}
		r := &connRegistration{c: c, done: make(chan struct{})}
		r.publication.Lock()
		r.submitErr = el.poller.Trigger(queue.HighPriority, el.register, r)
		r.publication.Unlock()
		if r.submitErr != nil {
			t.Fatal(r.submitErr)
		}
		select {
		case <-r.done:
			t.Fatal("admitted request completed before Polling or retirement")
		default:
		}
		started := make(chan struct{})
		result := make(chan error, 1)
		go func() {
			close(started)
			result <- r.awaitResult(el.poller.Done())
		}()
		<-started
		if err := el.poller.Close(); err != nil {
			t.Fatal(err)
		}
		if err := pr766LifecycleWait(t, result); !errors.Is(err, errorx.ErrEngineShutdown) {
			t.Fatalf("awaitResult after real retirement = %v, want ErrEngineShutdown", err)
		}
		// No discard call exists in this test. The production wait branch
		// must finalize the request and release its real duplicated socket.
		select {
		case <-r.done:
		default:
			t.Fatal("awaitResult returned without terminal request publication")
		}
		pr766LifecycleAssertClosed(t, fd)
		pr766LifecycleReuse(t, raw, fd, func() {
			if err := r.awaitResult(el.poller.Done()); !errors.Is(err, errorx.ErrEngineShutdown) {
				t.Fatalf("repeated awaitResult = %v, want original terminal error", err)
			}
		})
		if h.opens.Load() != 0 || h.closes.Load() != 0 || el.countConn() != 0 {
			t.Fatal("retired request registered a connection or invoked handlers")
		}
	})
	t.Run("completion_wins_both_ready", func(t *testing.T) {
		el, _ := pr766LifecycleLoop(t)
		raw := pr766v2TCPPair(t)
		fd := pr766LifecycleDup(t, raw, 0)
		t.Cleanup(func() {
			if err := unix.Close(fd); err != nil {
				t.Error(err)
			}
		})
		r := &connRegistration{c: &conn{fd: fd, loop: el}, done: make(chan struct{})}
		// Completion transfers ownership to this test's cleanup. Retirement
		// must preserve that already-published success and its descriptor.
		r.complete(nil)
		if err := el.poller.Close(); err != nil {
			t.Fatal(err)
		}
		if err := r.awaitResult(el.poller.Done()); err != nil {
			t.Fatalf("completed request with both channels ready = %v, want nil", err)
		}
		if _, err := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0); err != nil {
			t.Fatalf("successful completed request lost its retained descriptor: %v", err)
		}
	})
}
