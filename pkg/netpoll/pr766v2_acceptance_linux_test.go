//go:build linux

package netpoll

import (
	"errors"
	"sync/atomic"
	"testing"

	"golang.org/x/sys/unix"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/queue"
)

func pr766v2OpenPoller(t *testing.T) (*Poller, func()) {
	t.Helper()
	p, err := OpenPoller()
	if err != nil {
		t.Fatal(err)
	}
	closed := false
	closeOnce := func() {
		t.Helper()
		if !closed {
			closed = true
			if err := p.Close(); err != nil {
				t.Errorf("first Close = %v, want nil", err)
			}
		}
	}
	t.Cleanup(closeOnce)
	return p, closeOnce
}

func TestPR766V2CloseBeforePollingClosesDone(t *testing.T) {
	p, closePoller := pr766v2OpenPoller(t)
	select {
	case <-p.Done():
		t.Fatal("new poller already has closed Done")
	default:
	}
	closePoller()
	select {
	case <-p.Done():
	default:
		t.Fatal("Close returned before Polling ever started but Done remains open")
	}
}

func TestPR766V2TriggerAfterCloseRejects(t *testing.T) {
	p, closePoller := pr766v2OpenPoller(t)
	closePoller()
	var ran atomic.Bool
	for _, priority := range []queue.EventPriority{queue.HighPriority, queue.LowPriority} {
		err := p.Trigger(priority, func(any) error { ran.Store(true); return nil }, nil)
		if !errors.Is(err, errorx.ErrEngineShutdown) {
			t.Errorf("Trigger(%v) after Close = %v, want ErrEngineShutdown", priority, err)
		}
	}
	if ran.Load() {
		t.Error("rejected callback executed")
	}
	pr766v2AssertEmptyQueues(t, p)
}

func pr766v2AssertEmptyQueues(t *testing.T, p *Poller) {
	t.Helper()
	for _, pending := range []queue.AsyncTaskQueue{p.urgentAsyncTaskQueue, p.asyncTaskQueue} {
		count := 0
		// No Polling or producers exist: Dequeue is an exact terminal oracle,
		// unlike a transient Length observation. Also release baseline leftovers.
		for task := pending.Dequeue(); task != nil; task = pending.Dequeue() {
			count++
			queue.PutTask(task)
		}
		if count != 0 {
			t.Errorf("closed poller retained %d tasks in a queue, want 0", count)
		}
	}
}

func TestPR766V2CloseBeforePollingDropsAdmittedTasks(t *testing.T) {
	p, closePoller := pr766v2OpenPoller(t)
	var ran atomic.Bool
	for _, priority := range []queue.EventPriority{queue.HighPriority, queue.LowPriority} {
		if err := p.Trigger(priority, func(any) error { ran.Store(true); return nil }, nil); err != nil {
			t.Fatalf("Trigger before Close = %v, want nil", err)
		}
	}
	closePoller()
	if ran.Load() {
		t.Error("Close executed a callback without Polling")
	}
	pr766v2AssertEmptyQueues(t, p)
}

func TestPR766V2RepeatedClosePreservesReusedDescriptor(t *testing.T) {
	// F_DUPFD_CLOEXEC allocates, never overwrites an unrelated descriptor.
	// Keep the source open before creating the poller so it cannot occupy fd.
	source, err := unix.Eventfd(0, unix.EFD_CLOEXEC|unix.EFD_NONBLOCK)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := unix.Close(source); err != nil {
			t.Error(err)
		}
	})
	p, closePoller := pr766v2OpenPoller(t)
	originalFD := p.fd
	closePoller()
	reused, err := unix.FcntlInt(uintptr(source), unix.F_DUPFD_CLOEXEC, originalFD)
	if err != nil {
		t.Fatal(err)
	}
	alive := true
	t.Cleanup(func() {
		if alive {
			if err := unix.Close(reused); err != nil {
				t.Error(err)
			}
		}
	})
	if reused != originalFD {
		t.Fatalf("kernel allocated fd %d instead of closed poller fd %d; reuse precondition not established", reused, originalFD)
	}
	if _, err := unix.FcntlInt(uintptr(reused), unix.F_GETFD, 0); err != nil {
		t.Fatalf("reuse witness is not live before repeated Close: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Errorf("repeated Close = %v, want nil", err)
	}
	if _, err := unix.FcntlInt(uintptr(reused), unix.F_GETFD, 0); err != nil {
		alive = !errors.Is(err, unix.EBADF)
		t.Fatalf("repeated Close destroyed reused descriptor %d: %v", reused, err)
	}
	if err := unix.Close(reused); err != nil {
		t.Fatal(err)
	}
	alive = false
	if _, err := unix.FcntlInt(uintptr(reused), unix.F_GETFD, 0); !errors.Is(err, unix.EBADF) {
		t.Fatalf("closed witness F_GETFD = %v, want EBADF", err)
	}
}
