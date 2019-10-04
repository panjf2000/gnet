// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2017 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build linux

package netpoll

import (
	"log"

	"github.com/panjf2000/gnet/internal"
	"golang.org/x/sys/unix"
)

// Poller ...
type Poller struct {
	fd            int    // epoll fd
	wfd           int    // wake fd
	wfdBuf        []byte // wfd buffer to read packet
	asyncJobQueue internal.AsyncJobQueue
}

// OpenPoller ...
func OpenPoller() (*Poller, error) {
	poller := new(Poller)
	epollFD, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}
	poller.fd = epollFD
	r0, _, errno := unix.Syscall(unix.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		_ = unix.Close(epollFD)
		return nil, errno
	}
	poller.wfd = int(r0)
	poller.wfdBuf = make([]byte, 8)
	if err = poller.AddRead(poller.wfd); err != nil {
		return nil, err
	}
	poller.asyncJobQueue = internal.NewAsyncJobQueue()
	return poller, nil
}

// Close ...
func (p *Poller) Close() error {
	if err := unix.Close(p.wfd); err != nil {
		return err
	}
	return unix.Close(p.fd)
}

// Trigger ...
func (p *Poller) Trigger(job internal.Job) error {
	p.asyncJobQueue.Push(job)
	_, err := unix.Write(p.wfd, []byte{0, 0, 0, 0, 0, 0, 0, 1})
	return err
}

// Polling ...
func (p *Poller) Polling(callback func(fd int, ev uint32, job internal.Job) error) error {
	el := newEventList(initEvents)
	var wakenUp bool
	for {
		n, err := unix.EpollWait(p.fd, el.events, -1)
		if err != nil && err != unix.EINTR {
			log.Println(err)
			continue
		}
		for i := 0; i < n; i++ {
			if fd := int(el.events[i].Fd); fd != p.wfd {
				if err := callback(fd, el.events[i].Events, nil); err != nil {
					return err
				}
			} else {
				wakenUp = true
				if _, err := unix.Read(p.wfd, p.wfdBuf); err != nil {
					log.Println(err)
					continue
				}
			}
		}
		if wakenUp {
			wakenUp = false
			if err := p.asyncJobQueue.ForEach(func(job internal.Job) error {
				return callback(0, 0, job)
			}); err != nil {
				return err
			}
		}
		if n == el.size {
			el.increase()
		}
	}
}

// AddReadWrite ...
func (p *Poller) AddReadWrite(fd int) error {
	return unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd),
		Events: unix.EPOLLIN | unix.EPOLLOUT})
}

// AddRead ...
func (p *Poller) AddRead(fd int) error {
	return unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: unix.EPOLLIN})
}

// AddWrite ...
func (p *Poller) AddWrite(fd int) error {
	return unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: unix.EPOLLOUT})
}

// ModRead ...
func (p *Poller) ModRead(fd int) error {
	return unix.EpollCtl(p.fd, unix.EPOLL_CTL_MOD, fd, &unix.EpollEvent{Fd: int32(fd), Events: unix.EPOLLIN})
}

// ModReadWrite ...
func (p *Poller) ModReadWrite(fd int) error {
	return unix.EpollCtl(p.fd, unix.EPOLL_CTL_MOD, fd, &unix.EpollEvent{Fd: int32(fd),
		Events: unix.EPOLLIN | unix.EPOLLOUT})
}

// Delete ...
func (p *Poller) Delete(fd int) error {
	return unix.EpollCtl(p.fd, unix.EPOLL_CTL_DEL, fd, nil)
}
