// Copyright (c) 2021 The Gnet Authors. All rights reserved.
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

//go:build (freebsd || dragonfly || netbsd || openbsd || darwin) && poll_opt
// +build freebsd dragonfly netbsd openbsd darwin
// +build poll_opt

package netpoll

import (
	"os"
	"runtime"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/internal/queue"
	"github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

// Poller represents a poller which is in charge of monitoring file-descriptors.
type Poller struct {
	fd                          int
	wakeupCall                  int32
	asyncTaskQueue              queue.AsyncTaskQueue // queue with low priority
	urgentAsyncTaskQueue        queue.AsyncTaskQueue // queue with high priority
	highPriorityEventsThreshold int32                // threshold of high-priority events
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
	poller.urgentAsyncTaskQueue = queue.NewLockFreeQueue()
	poller.highPriorityEventsThreshold = MaxPollEventsCap
	return
}

// Close closes the poller.
func (p *Poller) Close() error {
	return os.NewSyscallError("close", unix.Close(p.fd))
}

var note = []unix.Kevent_t{{
	Ident:  0,
	Filter: unix.EVFILT_USER,
	Fflags: unix.NOTE_TRIGGER,
}}

// Trigger enqueues task and wakes up the poller to process pending tasks.
// By default, any incoming task will enqueued into urgentAsyncTaskQueue
// before the threshold of high-priority events is reached. When it happens,
// any asks other than high-priority tasks will be shunted to asyncTaskQueue.
//
// Note that asyncTaskQueue is a queue of low-priority whose size may grow large and tasks in it may backlog.
func (p *Poller) Trigger(priority queue.EventPriority, fn queue.TaskFunc, arg interface{}) (err error) {
	task := queue.GetTask()
	task.Run, task.Arg = fn, arg
	if priority > queue.HighPriority && p.urgentAsyncTaskQueue.Length() >= p.highPriorityEventsThreshold {
		p.asyncTaskQueue.Enqueue(task)
	} else {
		// There might be some low-priority tasks overflowing into urgentAsyncTaskQueue in a flash,
		// but that's tolerable because it ought to be a rare case.
		p.urgentAsyncTaskQueue.Enqueue(task)
	}
	if atomic.CompareAndSwapInt32(&p.wakeupCall, 0, 1) {
		if _, err = unix.Kevent(p.fd, note, nil, nil); err == unix.EAGAIN {
			err = nil
		}
	}
	return os.NewSyscallError("kevent trigger", err)
}

// Polling blocks the current goroutine, waiting for network-events.
func (p *Poller) Polling() error {
	el := newEventList(InitPollEventsCap)

	var (
		ts       unix.Timespec
		tsp      *unix.Timespec
		doChores bool
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

		for i := 0; i < n; i++ {
			ev := &el.events[i]
			if ev.Ident == 0 { // poller is awakened to run tasks in queues
				doChores = true
			} else {
				pollAttachment := (*PollAttachment)(unsafe.Pointer(ev.Udata))
				switch err = pollAttachment.Callback(int(ev.Ident), ev.Filter, ev.Flags); err {
				case nil:
				case errors.ErrAcceptSocket, errors.ErrEngineShutdown:
					return err
				default:
					logging.Warnf("error occurs in event-loop: %v", err)
				}
			}
		}

		if doChores {
			doChores = false
			task := p.urgentAsyncTaskQueue.Dequeue()
			for ; task != nil; task = p.urgentAsyncTaskQueue.Dequeue() {
				switch err = task.Run(task.Arg); err {
				case nil:
				case errors.ErrEngineShutdown:
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
				case errors.ErrEngineShutdown:
					return err
				default:
					logging.Warnf("error occurs in user-defined function, %v", err)
				}
				queue.PutTask(task)
			}
			atomic.StoreInt32(&p.wakeupCall, 0)
			if (!p.asyncTaskQueue.IsEmpty() || !p.urgentAsyncTaskQueue.IsEmpty()) && atomic.CompareAndSwapInt32(&p.wakeupCall, 0, 1) {
				switch _, err = unix.Kevent(p.fd, note, nil, nil); err {
				case nil, unix.EAGAIN:
				default:
					doChores = true
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
	var evs [2]unix.Kevent_t
	evs[0].Ident = keventIdent(pa.FD)
	evs[0].Flags = unix.EV_ADD
	evs[0].Filter = unix.EVFILT_READ
	evs[0].Udata = (*byte)(unsafe.Pointer(pa))
	evs[1] = evs[0]
	evs[1].Filter = unix.EVFILT_WRITE
	_, err := unix.Kevent(p.fd, evs[:], nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// AddRead registers the given file-descriptor with readable event to the poller.
func (p *Poller) AddRead(pa *PollAttachment) error {
	var evs [1]unix.Kevent_t
	evs[0].Ident = keventIdent(pa.FD)
	evs[0].Flags = unix.EV_ADD
	evs[0].Filter = unix.EVFILT_READ
	evs[0].Udata = (*byte)(unsafe.Pointer(pa))
	_, err := unix.Kevent(p.fd, evs[:], nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// AddWrite registers the given file-descriptor with writable event to the poller.
func (p *Poller) AddWrite(pa *PollAttachment) error {
	var evs [1]unix.Kevent_t
	evs[0].Ident = keventIdent(pa.FD)
	evs[0].Flags = unix.EV_ADD
	evs[0].Filter = unix.EVFILT_WRITE
	evs[0].Udata = (*byte)(unsafe.Pointer(pa))
	_, err := unix.Kevent(p.fd, evs[:], nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// ModRead renews the given file-descriptor with readable event in the poller.
func (p *Poller) ModRead(pa *PollAttachment) error {
	var evs [1]unix.Kevent_t
	evs[0].Ident = keventIdent(pa.FD)
	evs[0].Flags = unix.EV_DELETE
	evs[0].Filter = unix.EVFILT_WRITE
	evs[0].Udata = (*byte)(unsafe.Pointer(pa))
	_, err := unix.Kevent(p.fd, evs[:], nil, nil)
	return os.NewSyscallError("kevent delete", err)
}

// ModReadWrite renews the given file-descriptor with readable and writable events in the poller.
func (p *Poller) ModReadWrite(pa *PollAttachment) error {
	var evs [1]unix.Kevent_t
	evs[0].Ident = keventIdent(pa.FD)
	evs[0].Flags = unix.EV_ADD
	evs[0].Filter = unix.EVFILT_WRITE
	evs[0].Udata = (*byte)(unsafe.Pointer(pa))
	_, err := unix.Kevent(p.fd, evs[:], nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// Delete removes the given file-descriptor from the poller.
func (p *Poller) Delete(_ int) error {
	return nil
}
