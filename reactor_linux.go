// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build linux

package gnet

import "github.com/panjf2000/gnet/internal/netpoll"

func (svr *server) activateMainReactor() {
	defer svr.signalShutdown()

	_ = svr.mainLoop.poller.Polling(func(fd int, ev uint32) error {
		return svr.acceptNewConnection(fd)
	})
}

func (svr *server) activateSubReactor(lp *loop) {
	defer func() {
		if lp.idx == 0 && svr.opts.Ticker {
			close(svr.ticktock)
		}
		svr.signalShutdown()
	}()

	if lp.idx == 0 && svr.opts.Ticker {
		go lp.loopTicker()
	}

	_ = lp.poller.Polling(func(fd int, ev uint32) error {
		if c, ack := lp.connections[fd]; ack {
			switch c.outboundBuffer.IsEmpty() {
			// Don't change the ordering of processing EPOLLOUT | EPOLLRDHUP / EPOLLIN unless you're 100%
			// sure what you're doing!
			// Re-ordering can easily introduce bugs and bad side-effects, as I found out painfully in the past.
			case false:
				if ev&netpoll.OutEvents != 0 {
					return lp.loopOut(c)
				}
				return lp.loopIn(c)
			case true:
				if ev&netpoll.InEvents != 0 {
					return lp.loopIn(c)
				}
				return nil
			}
		}
		return nil
	})
}
