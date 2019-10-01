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
		lp := svr.loopGroup.next()
		c := &conn{
			fd:     nfd,
			sa:     sa,
			loop:   lp,
			inBuf:  ringbuffer.New(connRingBufferSize),
			outBuf: ringbuffer.New(connRingBufferSize),
		}
		_ = lp.loopOpened(c)
		_ = lp.poller.Trigger(func() (err error) {
			if err = lp.poller.AddRead(nfd); err == nil {
				lp.connections[nfd] = c
				return
			}
			return
		})
		return nil
	})
}

func (svr *server) activateSubReactor(lp *loop) {
	defer svr.signalShutdown()

	if lp.idx == 0 && svr.opts.Ticker {
		go lp.loopTicker()
	}

	_ = lp.poller.Polling(func(fd int, job internal.Job) error {
		if fd == 0 {
			return job()
		}
		c := lp.connections[fd]
		if c.outBuf.IsEmpty() {
			return lp.loopRead(c)
		}
		return lp.loopWrite(c)
	})
}
