// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build darwin netbsd freebsd openbsd dragonfly

package gnet

import (
	"github.com/panjf2000/gnet/internal"
	"github.com/panjf2000/gnet/netpoll"
)

func (svr *server) activateMainReactor() {
	defer svr.signalShutdown()

	_ = svr.mainLoop.poller.Polling(func(fd int, filter int16, job internal.Job) error {
		return svr.acceptNewConnection(fd)
	})
}

func (svr *server) activateSubReactor(lp *loop) {
	defer svr.signalShutdown()

	if lp.idx == 0 && svr.opts.Ticker {
		go lp.loopTicker()
	}

	_ = lp.poller.Polling(func(fd int, filter int16, job internal.Job) error {
		c := lp.connections[fd]
		switch {
		case !c.outboundBuffer.IsEmpty():
			if filter == netpoll.EVFilterWrite {
				return lp.loopOut(c)
			}
			return nil
		case filter == netpoll.EVFilterRead:
			return lp.loopIn(c)
		case filter == netpoll.EVFilterSock:
			return lp.loopCloseConn(c, nil)
		default:
			return nil
		}
	})
}
