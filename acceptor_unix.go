// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build linux darwin netbsd freebsd openbsd dragonfly

package gnet

import (
	"golang.org/x/sys/unix"
	"sync/atomic"
)

// 自定义权重
const (
	readWeight   = 3
	writeWeight  = 3
	acceptWeight = 4
	tickWeight   = 1
	upperBound   = 1 << 18
)

type LoadBalance int

const (
	Default LoadBalance = iota
	Priority
)

func (svr *server) acceptNewConnection(fd int) error {
	nfd, sa, err := unix.Accept(fd)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		return err
	}
	if err := unix.SetNonblock(nfd, true); err != nil {
		return err
	}
	el := svr.subLoopGroup.next()
	c := newTCPConn(nfd, el, sa)
	_ = el.poller.Trigger(func() (err error) {
		if err = el.poller.AddRead(nfd); err != nil {
			return
		}
		el.connections[nfd] = c
		err = el.loopOpen(c)
		return
	})
	return nil
}

func (svr *server) acceptNewPriConnection(fd int) error {
	nfd, sa, err := unix.Accept(fd)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		return err
	}
	if err := unix.SetNonblock(nfd, true); err != nil {
		return err
	}
	var (
		el          *eventloop
		minPriority int64 = 1 << 62
	)
	el = svr.subLoopGroup.next()
	svr.subLoopGroup.iterate(func(i int, e *eventloop) bool {
		var cur = atomic.LoadInt64(&e.priority)
		if cur < minPriority {
			el = e
			minPriority = cur
		}
		return true
	})
	c := newTCPConn(nfd, el, sa)
	_ = el.poller.Trigger(func() (err error) {
		if err = el.poller.AddRead(nfd); err != nil {
			return
		}
		el.connections[nfd] = c
		err = el.loopOpen(c)
		return
	})
	return nil
}
