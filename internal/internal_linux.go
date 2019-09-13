// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2017 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package internal

import "golang.org/x/sys/unix"

// Poller ...
type Poller struct {
	fd     int    // epoll fd
	wfd    int    // wake fd
	wfdBuf []byte // wfd buffer to read packet
	notes  noteQueue
}

// OpenPoller ...
func OpenPoller() *Poller {
	poller := new(Poller)
	epd, err := unix.EpollCreate1(0)
	if err != nil {
		panic(err)
	}
	poller.fd = epd
	r0, _, e0 := unix.Syscall(unix.SYS_EVENTFD2, 0, 0, 0)
	if e0 != 0 {
		_ = unix.Close(epd)
		panic(err)
	}
	poller.wfd = int(r0)
	poller.wfdBuf = make([]byte, 8)
	poller.AddRead(poller.wfd)
	return poller
}

// Close ...
func (p *Poller) Close() error {
	if err := unix.Close(p.wfd); err != nil {
		return err
	}
	return unix.Close(p.fd)
}

// Trigger ...
func (p *Poller) Trigger(note interface{}) error {
	p.notes.Add(note)
	_, err := unix.Write(p.wfd, []byte{0, 0, 0, 0, 0, 0, 0, 1})
	return err
}

// Polling ...
func (p *Poller) Polling(iter func(fd int, note interface{}) error) error {
	events := make([]unix.EpollEvent, 64)
	for {
		n, err := unix.EpollWait(p.fd, events, -1)
		if err != nil && err != unix.EINTR {
			return err
		}
		if err := p.notes.ForEach(func(note interface{}) error {
			return iter(0, note)
		}); err != nil {
			return err
		}
		for i := 0; i < n; i++ {
			if fd := int(events[i].Fd); fd != p.wfd {
				if err := iter(fd, nil); err != nil {
					return err
				}
			} else {
				if _, err := unix.Read(p.wfd, p.wfdBuf); err != nil {
					panic(err)
				}
			}
		}
	}
}

// AddReadWrite ...
func (p *Poller) AddReadWrite(fd int) {
	if err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd,
		&unix.EpollEvent{Fd: int32(fd),
			Events: unix.EPOLLIN | unix.EPOLLOUT,
		},
	); err != nil {
		panic(err)
	}
}

// AddRead ...
func (p *Poller) AddRead(fd int) {
	if err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd,
		&unix.EpollEvent{Fd: int32(fd),
			Events: unix.EPOLLIN,
		},
	); err != nil {
		panic(err)
	}
}

// ModRead ...
func (p *Poller) ModRead(fd int) {
	if err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_MOD, fd,
		&unix.EpollEvent{Fd: int32(fd),
			Events: unix.EPOLLIN,
		},
	); err != nil {
		panic(err)
	}
}

// ModReadWrite ...
func (p *Poller) ModReadWrite(fd int) {
	if err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_MOD, fd,
		&unix.EpollEvent{Fd: int32(fd),
			Events: unix.EPOLLIN | unix.EPOLLOUT,
		},
	); err != nil {
		panic(err)
	}
}

// ModDetach ...
func (p *Poller) ModDetach(fd int) {
	if err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_DEL, fd,
		&unix.EpollEvent{Fd: int32(fd),
			Events: unix.EPOLLIN | unix.EPOLLOUT,
		},
	); err != nil {
		panic(err)
	}
}
