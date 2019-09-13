// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2017 Joshua J Baker. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// +build darwin netbsd freebsd openbsd dragonfly

package internal

import "golang.org/x/sys/unix"

// Poller ...
type Poller struct {
	fd      int
	changes []unix.Kevent_t
	notes   noteQueue
}

// OpenPoller ...
func OpenPoller() *Poller {
	poller := new(Poller)
	kfd, err := unix.Kqueue()
	if err != nil {
		panic(err)
	}
	poller.fd = kfd
	_, err = unix.Kevent(poller.fd, []unix.Kevent_t{{
		Ident:  0,
		Filter: unix.EVFILT_USER,
		Flags:  unix.EV_ADD | unix.EV_CLEAR,
	}}, nil, nil)
	if err != nil {
		panic(err)
	}

	return poller
}

// Close ...
func (p *Poller) Close() error {
	return unix.Close(p.fd)
}

// Trigger ...
func (p *Poller) Trigger(note interface{}) error {
	p.notes.Add(note)
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{{
		Ident:  0,
		Filter: unix.EVFILT_USER,
		Fflags: unix.NOTE_TRIGGER,
	}}, nil, nil)
	return err
}

// Polling ...
func (p *Poller) Polling(iter func(fd int, note interface{}) error) error {
	events := make([]unix.Kevent_t, 128)
	for {
		n, err := unix.Kevent(p.fd, p.changes, events, nil)
		if err != nil && err != unix.EINTR {
			return err
		}
		p.changes = p.changes[:0]
		if err := p.notes.ForEach(func(note interface{}) error {
			return iter(0, note)
		}); err != nil {
			return err
		}
		for i := 0; i < n; i++ {
			if fd := int(events[i].Ident); fd != 0 {
				if err := iter(fd, nil); err != nil {
					return err
				}
			}
		}
	}
}

// AddRead ...
func (p *Poller) AddRead(fd int) {
	p.changes = append(p.changes,
		unix.Kevent_t{
			Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_READ,
		},
	)
}

// AddReadWrite ...
func (p *Poller) AddReadWrite(fd int) {
	p.changes = append(p.changes,
		unix.Kevent_t{
			Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_READ,
		},
		unix.Kevent_t{
			Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_WRITE,
		},
	)
}

// ModRead ...
func (p *Poller) ModRead(fd int) {
	p.changes = append(p.changes, unix.Kevent_t{
		Ident: uint64(fd), Flags: unix.EV_DELETE, Filter: unix.EVFILT_WRITE,
	})
}

// ModReadWrite ...
func (p *Poller) ModReadWrite(fd int) {
	p.changes = append(p.changes, unix.Kevent_t{
		Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_WRITE,
	})
}

// ModDetach ...
func (p *Poller) ModDetach(fd int) {
	p.changes = append(p.changes,
		unix.Kevent_t{
			Ident: uint64(fd), Flags: unix.EV_DELETE, Filter: unix.EVFILT_READ,
		},
		unix.Kevent_t{
			Ident: uint64(fd), Flags: unix.EV_DELETE, Filter: unix.EVFILT_WRITE,
		},
	)
}
