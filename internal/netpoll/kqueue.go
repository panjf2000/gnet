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

package netpoll

import (
	"os"
	"runtime"

	"github.com/panjf2000/gnet/errors"
	"github.com/panjf2000/gnet/internal"
	"github.com/panjf2000/gnet/internal/logging"
	"golang.org/x/sys/unix"
)

// Poller represents a poller which is in charge of monitoring file-descriptors.
type Poller struct {
	fd            int
	asyncJobQueue internal.AsyncJobQueue
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
	poller.asyncJobQueue = internal.NewAsyncJobQueue()
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

// Trigger wakes up the poller blocked in waiting for network-events and runs jobs in asyncJobQueue.
func (p *Poller) Trigger(job internal.Job) (err error) {
	if p.asyncJobQueue.Push(job) == 1 {
		_, err = unix.Kevent(p.fd, wakeChanges, nil, nil)
	}
	return os.NewSyscallError("kevent trigger", err)
}

// Polling blocks the current goroutine, waiting for network-events.
func (p *Poller) Polling(callback func(fd int, filter int16) error) error {
	el := newEventList(InitEvents)
	var wakenUp bool

	var (
		ts  unix.Timespec
		tsp *unix.Timespec
	)
	for {
		n, err := unix.Kevent(p.fd, nil, el.events, tsp)
		if n == 0 || (n < 0 && err == unix.EINTR) {
			tsp = nil
			runtime.Gosched()
			continue
		} else if err != nil {
			logging.DefaultLogger.Warnf("Error occurs in kqueue: %v", os.NewSyscallError("kevent wait", err))
			return err
		}
		tsp = &ts

		var evFilter int16
		for i := 0; i < n; i++ {
			if fd := int(el.events[i].Ident); fd != 0 {
				evFilter = el.events[i].Filter
				if (el.events[i].Flags&unix.EV_EOF != 0) || (el.events[i].Flags&unix.EV_ERROR != 0) {
					evFilter = EVFilterSock
				}
				switch err = callback(fd, evFilter); err {
				case nil:
				case errors.ErrAcceptSocket, errors.ErrServerShutdown:
					return err
				default:
					logging.DefaultLogger.Warnf("Error occurs in event-loop: %v", err)
				}
			} else {
				wakenUp = true
			}
		}

		if wakenUp {
			wakenUp = false
			leftover, err := p.asyncJobQueue.ForEach()
			switch err {
			case nil:
			case errors.ErrServerShutdown:
				return err
			default:
				if q := len(leftover); q > 0 && q == p.asyncJobQueue.Batch(leftover) {
					_, err = unix.Kevent(p.fd, wakeChanges, nil, nil)
				}
				logging.DefaultLogger.Warnf("Error occurs in user-defined function, %v", err)
			}
		}

		if n == el.size {
			el.increase()
		}
	}
}

// AddReadWrite registers the given file-descriptor with readable and writable events to the poller.
func (p *Poller) AddReadWrite(fd int) error {
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_READ},
		{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_WRITE},
	}, nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// AddRead registers the given file-descriptor with readable event to the poller.
func (p *Poller) AddRead(fd int) error {
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_READ},
	}, nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// AddWrite registers the given file-descriptor with writable event to the poller.
func (p *Poller) AddWrite(fd int) error {
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_WRITE},
	}, nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// ModRead renews the given file-descriptor with readable event in the poller.
func (p *Poller) ModRead(fd int) error {
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: uint64(fd), Flags: unix.EV_DELETE, Filter: unix.EVFILT_WRITE},
	}, nil, nil)
	return os.NewSyscallError("kevent delete", err)
}

// ModReadWrite renews the given file-descriptor with readable and writable events in the poller.
func (p *Poller) ModReadWrite(fd int) error {
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_WRITE},
	}, nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// Delete removes the given file-descriptor from the poller.
func (p *Poller) Delete(fd int) error {
	return nil
}
