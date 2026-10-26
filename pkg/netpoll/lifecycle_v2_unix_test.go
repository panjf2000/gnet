//go:build linux

package netpoll

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/queue"
)

func pr766LifecycleResult(t *testing.T, result <-chan error) error {
	t.Helper()
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	select {
	case err := <-result:
		return err
	case <-timer.C:
		t.Fatal("poller operation did not finish")
		return nil
	}
}

func pr766LifecycleFDAlive(t *testing.T, fd int) {
	t.Helper()
	if _, err := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0); err != nil {
		t.Fatalf("live descriptor %d: %v", fd, err)
	}
}

func TestPR766LifecycleCloseActivePreservesTasksAndDescriptors(t *testing.T) {
	p, closePoller := pr766v2OpenPoller(t)
	epollFD, wakeFD := p.fd, pr766LifecycleWakeFD(p)
	entered := make(chan error, 1)
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	if err := p.Trigger(queue.HighPriority, func(any) error {
		// Close from the callback must also be non-blocking and return EBUSY.
		entered <- p.Close()
		<-release
		return nil
	}, nil); err != nil {
		t.Fatal(err)
	}
	joined := make(chan error, 1)
	go func() { joined <- pr766LifecyclePoll(p) }()
	joinedAlready := false
	t.Cleanup(func() {
		unblock()
		if !joinedAlready {
			if err := p.Trigger(queue.HighPriority, func(any) error { return errorx.ErrEngineShutdown }, nil); err != nil {
				t.Error(err)
			}
			if err := pr766LifecycleResult(t, joined); !errors.Is(err, errorx.ErrEngineShutdown) {
				t.Error(err)
			}
		}
	})
	if err := pr766LifecycleResult(t, entered); !errors.Is(err, unix.EBUSY) {
		t.Fatalf("Close in active callback = %v, want EBUSY", err)
	}
	var retained atomic.Bool
	if err := p.Trigger(queue.LowPriority, func(any) error { retained.Store(true); return nil }, nil); err != nil {
		t.Fatal(err)
	}
	if err := p.Close(); !errors.Is(err, unix.EBUSY) {
		t.Fatalf("Close while callback held = %v, want EBUSY", err)
	}
	// A second Polling must not retire the first poller's queues on rejection.
	second := make(chan error, 1)
	go func() { second <- pr766LifecyclePoll(p) }()
	if err := pr766LifecycleResult(t, second); !errors.Is(err, unix.EBUSY) {
		t.Fatalf("second Polling = %v, want EBUSY", err)
	}
	select {
	case <-p.Done():
		t.Fatal("busy Close or rejected Polling closed Done")
	default:
	}
	pr766LifecycleFDAlive(t, epollFD)
	pr766LifecycleFDAlive(t, wakeFD)
	if err := p.Trigger(queue.HighPriority, func(any) error { return errorx.ErrEngineShutdown }, nil); err != nil {
		t.Fatal(err)
	}
	unblock()
	err := pr766LifecycleResult(t, joined)
	joinedAlready = true
	if !errors.Is(err, errorx.ErrEngineShutdown) {
		t.Fatalf("Polling after release = %v", err)
	}
	if !retained.Load() {
		t.Fatal("busy Close drained or lost an admitted callback")
	}
	select {
	case <-p.Done():
	default:
		t.Fatal("joined Polling left Done open")
	}
	// Retirement is deliberately not physical close.
	pr766LifecycleFDAlive(t, epollFD)
	pr766LifecycleFDAlive(t, wakeFD)
	closePoller()
	for _, fd := range []int{epollFD, wakeFD} {
		if _, err := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0); !errors.Is(err, unix.EBADF) {
			t.Errorf("descriptor %d after joined Close = %v, want EBADF", fd, err)
		}
	}
}

type pr766LifecycleGatedQueue struct {
	queue.AsyncTaskQueue
	entered chan struct{}
	release <-chan struct{}
}

func (q *pr766LifecycleGatedQueue) Enqueue(task *queue.Task) {
	q.AsyncTaskQueue.Enqueue(task)
	close(q.entered)
	<-q.release
}

func TestPR766LifecycleTriggerGateOrdersWakeBeforeClose(t *testing.T) {
	p, closePoller := pr766v2OpenPoller(t)
	epollFD, wakeFD := p.fd, pr766LifecycleWakeFD(p)
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	gate := &pr766LifecycleGatedQueue{AsyncTaskQueue: p.urgentAsyncTaskQueue, entered: make(chan struct{}), release: release}
	p.urgentAsyncTaskQueue = gate
	triggered := make(chan error, 1)
	go func() { triggered <- p.Trigger(queue.HighPriority, func(any) error { return nil }, nil) }()
	t.Cleanup(unblock)
	select {
	case <-gate.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("Trigger did not reach gated insertion")
	}
	// The queue barrier is an actual point within Trigger, without a global
	// hook. TryLock gives a synchronous negative control for the shared gate.
	if p.retireMu.TryLock() {
		p.retireMu.Unlock()
		t.Fatal("Trigger did not retain the lifecycle gate during admission")
	}
	closeStarted := make(chan struct{})
	closed := make(chan error, 1)
	go func() { close(closeStarted); closed <- p.Close() }()
	<-closeStarted
	pr766LifecycleFDAlive(t, epollFD)
	pr766LifecycleFDAlive(t, wakeFD)
	select {
	case err := <-closed:
		t.Fatalf("Close crossed a held producer gate: %v", err)
	default:
	}
	unblock()
	if err := pr766LifecycleResult(t, triggered); err != nil {
		t.Fatalf("admitted Trigger wakeup = %v, want nil before physical close", err)
	}
	if err := pr766LifecycleResult(t, closed); err != nil {
		t.Fatal(err)
	}
	closePoller()
	pr766v2AssertEmptyQueues(t, p)
	if err := p.Trigger(queue.HighPriority, func(any) error { return nil }, nil); !errors.Is(err, errorx.ErrEngineShutdown) {
		t.Fatalf("Trigger after gated Close = %v, want ErrEngineShutdown", err)
	}
}

func TestPR766LifecycleSaturatedWakeIsPendingEvent(t *testing.T) {
	p, closePoller := pr766v2OpenPoller(t)
	wakeFD := pr766LifecycleWakeFD(p)
	// Read native-endian one from the kernel, independent of package b.
	seed, err := unix.Eventfd(1, unix.EFD_CLOEXEC|unix.EFD_NONBLOCK)
	if err != nil {
		t.Fatal(err)
	}
	var probe [8]byte
	n, readErr := unix.Read(seed, probe[:])
	closeErr := unix.Close(seed)
	if readErr != nil || closeErr != nil || n != 8 {
		t.Fatalf("native eventfd seed: read=(%d, %v), close=%v", n, readErr, closeErr)
	}
	if n, err := unix.Write(wakeFD, probe[:]); err != nil || n != 8 {
		t.Fatalf("eventfd write control = (%d, %v)", n, err)
	}
	var readback [8]byte
	if n, err := unix.Read(wakeFD, readback[:]); err != nil || n != 8 || readback != probe {
		t.Fatalf("eventfd roundtrip = (%d, %v, %v), want exact probe", n, err, readback)
	}
	// The following EAGAIN is the independent kernel saturation oracle.
	var full [8]byte
	for i := range full {
		full[i] = 0xff
	}
	oneIndex := 0
	if probe[0] != 1 {
		oneIndex = 7
	}
	full[oneIndex] = 0xfe // UINT64_MAX-1 is the maximum eventfd counter.
	if n, err := unix.Write(wakeFD, full[:]); err != nil || n != 8 {
		t.Fatalf("saturate eventfd = (%d, %v)", n, err)
	}
	if _, err := unix.Write(wakeFD, probe[:]); !errors.Is(err, unix.EAGAIN) {
		t.Fatalf("negative full-eventfd control = %v, want EAGAIN", err)
	}
	var ran atomic.Bool
	if err := p.Trigger(queue.HighPriority, func(any) error { ran.Store(true); return errorx.ErrEngineShutdown }, nil); err != nil {
		t.Fatalf("Trigger with pending saturated wake = %v, want nil", err)
	}
	joined := make(chan error, 1)
	go func() { joined <- pr766LifecyclePoll(p) }()
	if err := pr766LifecycleResult(t, joined); !errors.Is(err, errorx.ErrEngineShutdown) {
		t.Fatalf("Polling saturated event = %v", err)
	}
	if !ran.Load() {
		t.Fatal("pending wake did not run accepted callback")
	}
	closePoller()
}
