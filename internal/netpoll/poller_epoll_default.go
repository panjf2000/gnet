// Copyright (c) 2019 Andy Pan
// Copyright (c) 2017 Joshua J Baker
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

//go:build linux && !poll_opt
// +build linux,!poll_opt

package netpoll

import (
	"errors"
	"os"
	"runtime"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/internal/queue"
	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

// Poller represents a poller which is in charge of monitoring file-descriptors.
type Poller struct {
	fd                          int    // epoll fd
	efd                         int    // eventfd
	efdBuf                      []byte // efd buffer to read an 8-byte integer
	wakeupCall                  int32
	asyncTaskQueue              queue.AsyncTaskQueue // queue with low priority
	urgentAsyncTaskQueue        queue.AsyncTaskQueue // queue with high priority
	highPriorityEventsThreshold int32                // threshold of high-priority events
}

// OpenPoller instantiates a poller.
func OpenPoller() (poller *Poller, err error) {
	poller = new(Poller)
	if poller.fd, err = unix.EpollCreate1(unix.EPOLL_CLOEXEC); err != nil {
		poller = nil
		err = os.NewSyscallError("epoll_create1", err)
		return
	}
	if poller.efd, err = unix.Eventfd(0, unix.EFD_NONBLOCK|unix.EFD_CLOEXEC); err != nil {
		_ = poller.Close()
		poller = nil
		err = os.NewSyscallError("eventfd", err)
		return
	}
	poller.efdBuf = make([]byte, 8)
	if err = poller.AddRead(&PollAttachment{FD: poller.efd}, true); err != nil {
		_ = poller.Close()
		poller = nil
		return
	}
	poller.asyncTaskQueue = queue.NewLockFreeQueue()
	poller.urgentAsyncTaskQueue = queue.NewLockFreeQueue()
	poller.highPriorityEventsThreshold = MaxPollEventsCap
	return
}

// Close closes the poller.
func (p *Poller) Close() error {
	_ = unix.Close(p.efd)
	return os.NewSyscallError("close", unix.Close(p.fd))
}

// Make the endianness of bytes compatible with more linux OSs under different processor-architectures,
// according to http://man7.org/linux/man-pages/man2/eventfd.2.html.
var (
	u uint64 = 1
	b        = (*(*[8]byte)(unsafe.Pointer(&u)))[:]
)

// Trigger enqueues task and wakes up the poller to process pending tasks.
// By default, any incoming task will enqueued into urgentAsyncTaskQueue
// before the threshold of high-priority events is reached. When it happens,
// any asks other than high-priority tasks will be shunted to asyncTaskQueue.
//
// Note that asyncTaskQueue is a queue of low-priority whose size may grow large and tasks in it may backlog.
func (p *Poller) Trigger(priority queue.EventPriority, fn queue.Func, param any) (err error) {
	task := queue.GetTask()
	task.Exec, task.Param = fn, param
	if priority > queue.HighPriority && p.urgentAsyncTaskQueue.Length() >= p.highPriorityEventsThreshold {
		p.asyncTaskQueue.Enqueue(task)
	} else {
		// There might be some low-priority tasks overflowing into urgentAsyncTaskQueue in a flash,
		// but that's tolerable because it ought to be a rare case.
		p.urgentAsyncTaskQueue.Enqueue(task)
	}
	if atomic.CompareAndSwapInt32(&p.wakeupCall, 0, 1) {
		for {
			_, err = unix.Write(p.efd, b)
			if err == unix.EAGAIN {
				_, _ = unix.Read(p.efd, p.efdBuf)
				continue
			}
			break
		}
	}
	return os.NewSyscallError("write", err)
}

// Polling blocks the current goroutine, waiting for network-events.
func (p *Poller) Polling(callback PollEventHandler) error {
	el := newEventList(InitPollEventsCap)
	var doChores bool

	msec := -1
	for {
		n, err := unix.EpollWait(p.fd, el.events, msec)
		if n == 0 || (n < 0 && err == unix.EINTR) {
			msec = -1
			runtime.Gosched()
			continue
		} else if err != nil {
			logging.Errorf("error occurs in epoll: %v", os.NewSyscallError("epoll_wait", err))
			return err
		}
		msec = 0

		for i := 0; i < n; i++ {
			ev := &el.events[i]
			if fd := int(ev.Fd); fd == p.efd { // poller is awakened to run tasks in queues.
				doChores = true
			} else {
				err = callback(fd, ev.Events, 0)
				if errors.Is(err, errorx.ErrAcceptSocket) || errors.Is(err, errorx.ErrEngineShutdown) {
					return err
				}
			}
		}

		if doChores {
			doChores = false
			task := p.urgentAsyncTaskQueue.Dequeue()
			for ; task != nil; task = p.urgentAsyncTaskQueue.Dequeue() {
				err = task.Exec(task.Param)
				if errors.Is(err, errorx.ErrEngineShutdown) {
					return err
				}
				queue.PutTask(task)
			}
			for i := 0; i < MaxAsyncTasksAtOneTime; i++ {
				if task = p.asyncTaskQueue.Dequeue(); task == nil {
					break
				}
				err = task.Exec(task.Param)
				if errors.Is(err, errorx.ErrEngineShutdown) {
					return err
				}
				queue.PutTask(task)
			}
			atomic.StoreInt32(&p.wakeupCall, 0)
			if (!p.asyncTaskQueue.IsEmpty() || !p.urgentAsyncTaskQueue.IsEmpty()) && atomic.CompareAndSwapInt32(&p.wakeupCall, 0, 1) {
				for {
					_, err = unix.Write(p.efd, b)
					if err == unix.EAGAIN {
						_, _ = unix.Read(p.efd, p.efdBuf)
						continue
					}
					if err != nil {
						logging.Errorf("failed to notify next round of event-loop for leftover tasks, %v", os.NewSyscallError("write", err))
					}
					break
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
func (p *Poller) AddReadWrite(pa *PollAttachment, edgeTriggered bool) error {
	var ev uint32 = ReadWriteEvents
	if edgeTriggered {
		ev |= unix.EPOLLET | unix.EPOLLRDHUP
	}
	return os.NewSyscallError("epoll_ctl add",
		unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, pa.FD, &unix.EpollEvent{Fd: int32(pa.FD), Events: ev}))
}

// AddRead registers the given file-descriptor with readable event to the poller.
func (p *Poller) AddRead(pa *PollAttachment, edgeTriggered bool) error {
	var ev uint32 = ReadEvents
	if edgeTriggered {
		ev |= unix.EPOLLET | unix.EPOLLRDHUP
	}
	return os.NewSyscallError("epoll_ctl add",
		unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, pa.FD, &unix.EpollEvent{Fd: int32(pa.FD), Events: ev}))
}

// AddWrite registers the given file-descriptor with writable event to the poller.
func (p *Poller) AddWrite(pa *PollAttachment, edgeTriggered bool) error {
	var ev uint32 = WriteEvents
	if edgeTriggered {
		ev |= unix.EPOLLET | unix.EPOLLRDHUP
	}
	return os.NewSyscallError("epoll_ctl add",
		unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, pa.FD, &unix.EpollEvent{Fd: int32(pa.FD), Events: ev}))
}

// ModRead renews the given file-descriptor with readable event in the poller.
func (p *Poller) ModRead(pa *PollAttachment, edgeTriggered bool) error {
	var ev uint32 = ReadEvents
	if edgeTriggered {
		ev |= unix.EPOLLET | unix.EPOLLRDHUP
	}
	return os.NewSyscallError("epoll_ctl mod",
		unix.EpollCtl(p.fd, unix.EPOLL_CTL_MOD, pa.FD, &unix.EpollEvent{Fd: int32(pa.FD), Events: ev}))
}

// ModReadWrite renews the given file-descriptor with readable and writable events in the poller.
func (p *Poller) ModReadWrite(pa *PollAttachment, edgeTriggered bool) error {
	var ev uint32 = ReadWriteEvents
	if edgeTriggered {
		ev |= unix.EPOLLET | unix.EPOLLRDHUP
	}
	return os.NewSyscallError("epoll_ctl mod",
		unix.EpollCtl(p.fd, unix.EPOLL_CTL_MOD, pa.FD, &unix.EpollEvent{Fd: int32(pa.FD), Events: ev}))
}

// Delete removes the given file-descriptor from the poller.
func (p *Poller) Delete(fd int) error {
	return os.NewSyscallError("epoll_ctl del", unix.EpollCtl(p.fd, unix.EPOLL_CTL_DEL, fd, nil))
}
