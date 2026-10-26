//go:build linux

package netpoll

import (
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/queue"
)

func pr766WakeNativeOne(t *testing.T) [8]byte {
	t.Helper()
	fd, err := unix.Eventfd(1, unix.EFD_NONBLOCK|unix.EFD_CLOEXEC)
	if err != nil {
		t.Fatal(err)
	}
	var one [8]byte
	n, readErr := unix.Read(fd, one[:])
	closeErr := unix.Close(fd)
	if n != 8 || readErr != nil || closeErr != nil {
		t.Fatalf("eventfd seed: read=(%d, %v), close=%v", n, readErr, closeErr)
	}
	return one
}

// pr766WakeStart always joins Polling, including when the test proves a lost
// edge. Recovery is deliberately outside the progress assertion: it drains the
// counter and writes a fresh kernel edge, after queuing a shutdown callback.
func pr766WakeStart(t *testing.T, p *Poller, one [8]byte) <-chan struct{} {
	t.Helper()
	finished := make(chan struct{})
	var pollErr error
	go func() {
		pollErr = pr766LifecyclePoll(p)
		close(finished)
	}()
	t.Cleanup(func() {
		select {
		case <-finished:
		default:
			if err := p.Trigger(queue.HighPriority, func(any) error { return errorx.ErrEngineShutdown }, nil); err != nil && !errors.Is(err, errorx.ErrEngineShutdown) {
				t.Errorf("cleanup shutdown submission: %v", err)
			}
			var drained [8]byte
			fd := pr766LifecycleWakeFD(p)
			for {
				_, err := unix.Read(fd, drained[:])
				if errors.Is(err, unix.EINTR) {
					continue
				}
				if err != nil && !errors.Is(err, unix.EAGAIN) {
					t.Errorf("cleanup drain: %v", err)
				}
				break
			}
			for {
				_, err := unix.Write(fd, one[:])
				if errors.Is(err, unix.EINTR) {
					continue
				}
				if err != nil {
					t.Errorf("cleanup manual wake: %v", err)
				}
				break
			}
			select {
			case <-finished:
				t.Log("cleanup joined Polling after an explicit eventfd drain and fresh wake")
			case <-time.After(3 * time.Second):
				t.Error("cleanup could not join Polling after manual wake")
				return
			}
		}
		if !errors.Is(pollErr, errorx.ErrEngineShutdown) {
			t.Errorf("joined Polling = %v, want ErrEngineShutdown", pollErr)
		}
	})
	return finished
}

func TestPR766WakeSaturationAfterConsumedEdgeMakesProgress(t *testing.T) {
	p, _ := pr766v2OpenPoller(t)
	one := pr766WakeNativeOne(t)
	var saturated [8]byte
	for i := range saturated {
		saturated[i] = 0xff
	}
	if one[0] == 1 {
		saturated[0] = 0xfe
	} else {
		saturated[7] = 0xfe
	}
	wakeFD := pr766LifecycleWakeFD(p)
	if n, err := unix.Write(wakeFD, saturated[:]); n != 8 || err != nil {
		t.Fatalf("eventfd saturation = (%d, %v)", n, err)
	}
	if _, err := unix.Write(wakeFD, one[:]); !errors.Is(err, unix.EAGAIN) {
		t.Fatalf("full-eventfd negative control = %v, want EAGAIN", err)
	}
	first := make(chan struct{})
	if err := p.Trigger(queue.HighPriority, func(any) error { close(first); return nil }, nil); err != nil {
		t.Fatal(err)
	}
	finished := pr766WakeStart(t, p, one)
	select {
	case <-first:
	case <-time.After(3 * time.Second):
		t.Fatal("first pending edge did not execute its callback")
	}
	// The callback has returned only once the loop resets its wake flag. Wait
	// for that actual boundary instead of guessing when epoll consumed the edge.
	deadline := time.Now().Add(3 * time.Second)
	for atomic.LoadInt32(&p.wakeupCall) != 0 {
		if time.Now().After(deadline) {
			t.Fatal("poller did not reset wakeupCall after the first callback")
		}
		runtime.Gosched()
	}
	second := make(chan struct{})
	if err := p.Trigger(queue.HighPriority, func(any) error {
		close(second)
		return errorx.ErrEngineShutdown
	}, nil); err != nil {
		t.Fatalf("second Trigger = %v, want progress or an explicit terminal error", err)
	}
	select {
	case <-second:
	case <-time.After(3 * time.Second):
		t.Fatal("second callback stalled: Trigger returned nil after the ET edge had already been consumed")
	}
	select {
	case <-finished:
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown callback ran but Polling did not return")
	}
}

func TestPR766WakeFatalWriteResetsAdmissionForNextWake(t *testing.T) {
	p, _ := pr766v2OpenPoller(t)
	one := pr766WakeNativeOne(t)
	wakeFD := pr766LifecycleWakeFD(p)
	_, controlErr := unix.Write(-1, one[:])
	if !errors.Is(controlErr, unix.EBADF) {
		t.Fatalf("negative write control = %v, want EBADF", controlErr)
	}
	order := make(chan int, 2)
	// No Polling exists yet. The injection is local to this poller and is
	// restored immediately after the real Trigger returns, even if it failed.
	pr766WakeSetFD(p, -1)
	err := p.Trigger(queue.HighPriority, func(any) error { order <- 1; return nil }, nil)
	pr766WakeSetFD(p, wakeFD)
	if !errors.Is(err, unix.EBADF) {
		t.Fatalf("Trigger on invalid wake descriptor = %v, want EBADF", err)
	}
	if got := atomic.LoadInt32(&p.wakeupCall); got != 0 {
		t.Fatalf("wake flag after fatal write = %d, want 0", got)
	}
	select {
	case value := <-order:
		t.Fatalf("failed wake executed callback before Polling: %d", value)
	default:
	}
	if err := p.Trigger(queue.HighPriority, func(any) error { order <- 2; return errorx.ErrEngineShutdown }, nil); err != nil {
		t.Fatalf("Trigger after restoring wake descriptor = %v", err)
	}
	finished := pr766WakeStart(t, p, one)
	select {
	case <-finished:
	case <-time.After(3 * time.Second):
		t.Fatal("successful retry wake did not progress after the fatal write")
	}
	for _, want := range []int{1, 2} {
		select {
		case got := <-order:
			if got != want {
				t.Fatalf("callback order = %d, want %d", got, want)
			}
		default:
			t.Fatalf("missing callback %d after joined Polling", want)
		}
	}
}
