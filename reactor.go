// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build darwin netbsd freebsd openbsd dragonfly linux

package gnet

import (
	"github.com/panjf2000/gnet/internal"
	"github.com/panjf2000/gnet/ringbuffer"
	"golang.org/x/sys/unix"
)

func (svr *server) activateMainReactor() {
	defer svr.signalShutdown()

	_ = svr.mainLoop.poller.Polling(func(fd int, job internal.Job) error {
		if fd == 0 {
			return job()
		}
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
		lp := svr.eventLoopGroup.next()
		conn := &conn{
			fd:     nfd,
			sa:     sa,
			loop:   lp,
			inBuf:  ringbuffer.New(connRingBufferSize),
			outBuf: ringbuffer.New(connRingBufferSize),
		}
		_ = lp.loopOpened(conn)
		_ = lp.poller.Trigger(func() (err error) {
			if err = lp.poller.AddRead(nfd); err == nil {
				lp.connections[nfd] = conn
				return
			}
			return
		})
		return nil
	})
}

func (svr *server) activateSubReactor(loop *loop) {
	defer svr.signalShutdown()

	if loop.idx == 0 && svr.events.Tick != nil {
		go loop.loopTicker()
	}

	_ = loop.poller.Polling(func(fd int, job internal.Job) error {
		if fd == 0 {
			return job()
		}
		conn := loop.connections[fd]
		if conn.outBuf.IsEmpty() {
			return loop.loopRead(conn)
		}
		return loop.loopWrite(conn)
	})
}
