// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build linux

package gnet

import (
	"github.com/panjf2000/gnet/internal"
	"github.com/panjf2000/gnet/netpoll"
)

func (svr *server) activateMainReactor() {
	defer svr.signalShutdown()

	_ = svr.mainLoop.poller.Polling(func(fd int, ev uint32, job internal.Job) error {
		if fd == 0 {
			return job()
		}
		return svr.acceptNewConnection(fd)
	})
}

func (svr *server) activateSubReactor(lp *loop) {
	defer svr.signalShutdown()

	if lp.idx == 0 && svr.opts.Ticker {
		go lp.loopTicker()
	}

	_ = lp.poller.Polling(func(fd int, ev uint32, job internal.Job) error {
		if fd == 0 {
			return job()
		}

		c := lp.connections[fd]
		if !c.outboundBuffer.IsEmpty() {
			if ev&netpoll.OutEvents != 0 {
				return lp.loopOut(c)
			}
		} else if ev&netpoll.InEvents != 0 {
			return lp.loopIn(c)
		}

		return nil
	})
}
