// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build darwin netbsd freebsd openbsd dragonfly

package gnet

import "github.com/panjf2000/gnet/internal/netpoll"

func (svr *server) activateMainReactor() {
	defer svr.signalShutdown()

	sniffError(svr.mainLoop.poller.Polling(func(fd int, filter int16) error {
		return svr.acceptNewConnection(fd)
	}))
}

func (svr *server) activateSubReactor(lp *loop) {
	defer svr.signalShutdown()

	if lp.idx == 0 && svr.opts.Ticker {
		go lp.loopTicker()
	}

	sniffError(lp.poller.Polling(func(fd int, filter int16) error {
		if c, ack := lp.connections[fd]; ack {
			if filter == netpoll.EVFilterSock {
				return lp.loopCloseConn(c, nil)
			}
			switch c.outboundBuffer.IsEmpty() {
			// Don't change the ordering of processing EVFILT_WRITE | EVFILT_READ | EV_ERROR/EV_EOF unless you're 100%
			// sure what you're doing!
			// Re-ordering can easily introduce bugs and bad side-effects, as I found out painfully in the past.
			case false:
				if filter == netpoll.EVFilterWrite {
					return lp.loopOut(c)
				}
				return nil
			case true:
				if filter == netpoll.EVFilterRead {
					return lp.loopIn(c)
				}
				return nil
			}
		}
		return nil
	}))
}
