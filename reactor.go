// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build darwin netbsd freebsd openbsd dragonfly linux

package gnet

import (
	"github.com/panjf2000/gnet/ringbuffer"
	"golang.org/x/sys/unix"
)

type socket struct {
	fd   int
	conn *conn
}

func (svr *server) activateMainReactor() {
	defer func() {
		svr.signalShutdown()
		svr.wg.Done()
	}()

	_ = svr.mainLoop.poller.Polling(func(fd int, note interface{}) error {
		if fd == 0 {
			return svr.mainLoop.loopNote(svr, note)
		}

		if svr.ln.pconn != nil {
			return svr.mainLoop.loopUDPRead(svr, fd)
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
		_ = lp.loopOpened(svr, conn)
		_ = lp.poller.Trigger(&socket{fd: nfd, conn: conn})
		return nil
	})
}

func (svr *server) activateSubReactor(loop *loop) {
	defer func() {
		svr.signalShutdown()
		svr.wg.Done()
	}()

	if loop.idx == 0 && svr.events.Tick != nil {
		go loop.loopTicker(svr)
	}

	_ = loop.poller.Polling(func(fd int, note interface{}) error {
		if fd == 0 {
			return loop.loopNote(svr, note)
		}

		if c := loop.connections[fd]; c.outBuf.Length() > 0 {
			return loop.loopWrite(svr, c)
		} else {
			return loop.loopRead(svr, c)
		}
	})
}
