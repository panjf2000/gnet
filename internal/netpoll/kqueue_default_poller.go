// Copyright (c) 2019 Andy Pan
// Copyright (c) 2017 Joshua J Baker
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// +build freebsd dragonfly darwin
// +build !poll_opt

package netpoll

import (
	"os"
	"runtime"
	"sync/atomic"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/errors"
	"github.com/panjf2000/gnet/internal/queue"
	"github.com/panjf2000/gnet/logging"
)

// Poller represents a poller which is in charge of monitoring file-descriptors.
type Poller struct {
	fd                  int
	netpollWakeSig      int32
	asyncTaskQueue      queue.AsyncTaskQueue // queue with low priority
	priorAsyncTaskQueue queue.AsyncTaskQueue // queue with high priority
}

// OpenPoller instantiates a poller.
func OpenPoller() (poller *Poller, err error) {
	poller = new(Poller)
	if poller.fd, err = unix.Kqueue(); err != nil {
		poller = nil
		err = os.NewSyscallError("kqueue", err)
		return
	}
	if _, err = unix.Kevent(poller.fd, []unix.Kevent_t{{
		Ident:  0,
		Filter: unix.EVFILT_USER,
		Flags:  unix.EV_ADD | unix.EV_CLEAR,
	}}, nil, nil); err != nil {
		_ = poller.Close()
		poller = nil
		err = os.NewSyscallError("kevent add|clear", err)
		return
	}
	poller.asyncTaskQueue = queue.NewLockFreeQueue()
	poller.priorAsyncTaskQueue = queue.NewLockFreeQueue()
	return
}

// Close closes the poller.
func (p *Poller) Close() error {
	return os.NewSyscallError("close", unix.Close(p.fd))
}

var wakeChanges = []unix.Kevent_t{{
	Ident:  0,
	Filter: unix.EVFILT_USER,
	Fflags: unix.NOTE_TRIGGER,
}}

// UrgentTrigger puts task into priorAsyncTaskQueue and wakes up the poller which is waiting for network-events,
// then the poller will get tasks from priorAsyncTaskQueue and run them.
//
// Note that priorAsyncTaskQueue is a queue with high-priority and its size is expected to be small,
// so only those urgent tasks should be put into this queue.
func (p *Poller) UrgentTrigger(fn queue.TaskFunc, arg interface{}) (err error) {
	task := queue.GetTask()
	task.Run, task.Arg = fn, arg
	p.priorAsyncTaskQueue.Enqueue(task)
	if atomic.CompareAndSwapInt32(&p.netpollWakeSig, 0, 1) {
		for _, err = unix.Kevent(p.fd, wakeChanges, nil, nil); err == unix.EINTR || err == unix.EAGAIN; _, err = unix.Kevent(p.fd, wakeChanges, nil, nil) {
		}
	}
	return os.NewSyscallError("kevent trigger", err)
}

// Trigger is like UrgentTrigger but it puts task into asyncTaskQueue,
// call this method when the task is not so urgent, for instance writing data back to client.
//
// Note that asyncTaskQueue is a queue with low-priority whose size may grow large and tasks in it may backlog.
func (p *Poller) Trigger(fn queue.TaskFunc, arg interface{}) (err error) {
	task := queue.GetTask()
	task.Run, task.Arg = fn, arg
	p.asyncTaskQueue.Enqueue(task)
	if atomic.CompareAndSwapInt32(&p.netpollWakeSig, 0, 1) {
		for _, err = unix.Kevent(p.fd, wakeChanges, nil, nil); err == unix.EINTR || err == unix.EAGAIN; _, err = unix.Kevent(p.fd, wakeChanges, nil, nil) {
		}
	}
	return os.NewSyscallError("kevent trigger", err)
}

// Polling blocks the current goroutine, waiting for network-events.
func (p *Poller) Polling(callback func(fd int, filter int16) error) error {
	el := newEventList(InitPollEventsCap)

	var (
		ts      unix.Timespec
		tsp     *unix.Timespec
		wakenUp bool
	)
	for {
		n, err := unix.Kevent(p.fd, nil, el.events, tsp)
		if n == 0 || (n < 0 && err == unix.EINTR) {
			tsp = nil
			runtime.Gosched()
			continue
		} else if err != nil {
			logging.Errorf("error occurs in kqueue: %v", os.NewSyscallError("kevent wait", err))
			return err
		}
		tsp = &ts

		var evFilter int16
		for i := 0; i < n; i++ {
			ev := &el.events[i]
			if fd := int(ev.Ident); fd != 0 {
				evFilter = el.events[i].Filter
				if (ev.Flags&unix.EV_EOF != 0) || (ev.Flags&unix.EV_ERROR != 0) {
					evFilter = EVFilterSock
				}
				switch err = callback(fd, evFilter); err {
				case nil:
				case errors.ErrAcceptSocket, errors.ErrServerShutdown:
					return err
				default:
					logging.Warnf("error occurs in event-loop: %v", err)
				}
			} else { // poller is awaken to run tasks in queues.
				wakenUp = true
			}
		}

		if wakenUp {
			wakenUp = false
			task := p.priorAsyncTaskQueue.Dequeue()
			for ; task != nil; task = p.priorAsyncTaskQueue.Dequeue() {
				switch err = task.Run(task.Arg); err {
				case nil:
				case errors.ErrServerShutdown:
					return err
				default:
					logging.Warnf("error occurs in user-defined function, %v", err)
				}
				queue.PutTask(task)
			}
			for i := 0; i < MaxAsyncTasksAtOneTime; i++ {
				if task = p.asyncTaskQueue.Dequeue(); task == nil {
					break
				}
				switch err = task.Run(task.Arg); err {
				case nil:
				case errors.ErrServerShutdown:
					return err
				default:
					logging.Warnf("error occurs in user-defined function, %v", err)
				}
				queue.PutTask(task)
			}
			atomic.StoreInt32(&p.netpollWakeSig, 0)
			if !p.asyncTaskQueue.Empty() || !p.priorAsyncTaskQueue.Empty() {
				for _, err = unix.Kevent(p.fd, wakeChanges, nil, nil); err == unix.EINTR || err == unix.EAGAIN; _, err = unix.Kevent(p.fd, wakeChanges, nil, nil) {
				}
			}
		}

		if n == el.size {
			el.expand()
		} else if n < el.size>>1 {
			el.shrink()
		}
	}
}

// AddReadWrite registers the given file-descriptor with readable and writable events to the poller.
func (p *Poller) AddReadWrite(pa *PollAttachment) error {
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: uint64(pa.FD), Flags: unix.EV_ADD, Filter: unix.EVFILT_READ},
		{Ident: uint64(pa.FD), Flags: unix.EV_ADD, Filter: unix.EVFILT_WRITE},
	}, nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// AddRead registers the given file-descriptor with readable event to the poller.
func (p *Poller) AddRead(pa *PollAttachment) error {
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: uint64(pa.FD), Flags: unix.EV_ADD, Filter: unix.EVFILT_READ},
	}, nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// AddWrite registers the given file-descriptor with writable event to the poller.
func (p *Poller) AddWrite(pa *PollAttachment) error {
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: uint64(pa.FD), Flags: unix.EV_ADD, Filter: unix.EVFILT_WRITE},
	}, nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// ModRead renews the given file-descriptor with readable event in the poller.
func (p *Poller) ModRead(pa *PollAttachment) error {
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: uint64(pa.FD), Flags: unix.EV_DELETE, Filter: unix.EVFILT_WRITE},
	}, nil, nil)
	return os.NewSyscallError("kevent delete", err)
}

// ModReadWrite renews the given file-descriptor with readable and writable events in the poller.
func (p *Poller) ModReadWrite(pa *PollAttachment) error {
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: uint64(pa.FD), Flags: unix.EV_ADD, Filter: unix.EVFILT_WRITE},
	}, nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// Delete removes the given file-descriptor from the poller.
func (p *Poller) Delete(_ int) error {
	return nil
}
