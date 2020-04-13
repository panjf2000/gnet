// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2017 Joshua J Baker. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// +build darwin netbsd freebsd openbsd dragonfly

package netpoll

import (
	"log"

	"github.com/panjf2000/gnet/internal"
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
		return
	}
	if _, err = unix.Kevent(poller.fd, []unix.Kevent_t{{
		Ident:  0,
		Filter: unix.EVFILT_USER,
		Flags:  unix.EV_ADD | unix.EV_CLEAR,
	}}, nil, nil); err != nil {
		_ = poller.Close()
		poller = nil
		return
	}
	poller.asyncJobQueue = internal.NewAsyncJobQueue()
	return
}

// Close closes the poller.
func (p *Poller) Close() error {
	return unix.Close(p.fd)
}

var wakeChanges = []unix.Kevent_t{{
	Ident:  0,
	Filter: unix.EVFILT_USER,
	Fflags: unix.NOTE_TRIGGER,
}}

// Trigger wakes up the poller blocked in waiting for network-events and runs jobs in asyncJobQueue.
func (p *Poller) Trigger(job internal.Job) error {
	if p.asyncJobQueue.Push(job) == 1 {
		_, err := unix.Kevent(p.fd, wakeChanges, nil, nil)
		return err
	}
	return nil
}

// Polling blocks the current goroutine, waiting for network-events.
func (p *Poller) Polling(callback func(fd int, filter int16) error) (err error) {
	el := newEventList(InitEvents)
	var wakenUp bool
	for {
		n, err0 := unix.Kevent(p.fd, nil, el.events, nil)
		if err0 != nil && err0 != unix.EINTR {
			log.Println(err0)
			continue
		}
		var evFilter int16
		for i := 0; i < n; i++ {
			if fd := int(el.events[i].Ident); fd != 0 {
				evFilter = el.events[i].Filter
				if (el.events[i].Flags&unix.EV_EOF != 0) || (el.events[i].Flags&unix.EV_ERROR != 0) {
					evFilter = EVFilterSock
				}
				if err = callback(fd, evFilter); err != nil {
					return
				}
			} else {
				wakenUp = true
			}
		}
		if wakenUp {
			wakenUp = false
			if err = p.asyncJobQueue.ForEach(); err != nil {
				return
			}
		}
		if n == el.size {
			el.increase()
		}
	}
}

// AddReadWrite registers the given file-descriptor with readable and writable events to the poller.
func (p *Poller) AddReadWrite(fd int) error {
	if _, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_READ},
		{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_WRITE},
	}, nil, nil); err != nil {
		return err
	}
	return nil
}

// AddRead registers the given file-descriptor with readable event to the poller.
func (p *Poller) AddRead(fd int) error {
	if _, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_READ}}, nil, nil); err != nil {
		return err
	}
	return nil
}

// AddWrite registers the given file-descriptor with writable event to the poller.
func (p *Poller) AddWrite(fd int) error {
	if _, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_WRITE}}, nil, nil); err != nil {
		return err
	}
	return nil
}

// ModRead renews the given file-descriptor with readable event in the poller.
func (p *Poller) ModRead(fd int) error {
	if _, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: uint64(fd), Flags: unix.EV_DELETE, Filter: unix.EVFILT_WRITE}}, nil, nil); err != nil {
		return err
	}
	return nil
}

// ModReadWrite renews the given file-descriptor with readable and writable events in the poller.
func (p *Poller) ModReadWrite(fd int) error {
	if _, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_WRITE}}, nil, nil); err != nil {
		return err
	}
	return nil
}

// Delete removes the given file-descriptor from the poller.
func (p *Poller) Delete(fd int) error {
	return nil
}
