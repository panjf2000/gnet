// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build linux

package gnet

import (
	"github.com/panjf2000/gnet/internal/netpoll"
	"sync/atomic"
	"time"
)

func (svr *server) activateMainReactor() {
	defer svr.signalShutdown()
	switch svr.opts.LoadBalance {
	case Default:
		svr.logger.Printf("main reactor exits with error:%v\n", svr.mainLoop.poller.Polling(func(fd int, ev uint32) error {
			return svr.acceptNewConnection(fd)
		}))
	case Priority:
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				<-ticker.C
				var flag = false
				svr.subLoopGroup.iterate(func(i int, e *eventloop) bool {
					if e.priority > upperBound {
						flag = true
					}
					return true
				})
				if flag {
					svr.subLoopGroup.iterate(func(i int, e *eventloop) bool {
						var p = atomic.LoadInt64(&e.priority)
						p >>= 2
						if p < 0 {
							p = 0
						}
						atomic.StoreInt64(&e.priority, p)
						return true
					})
				}
			}
		}()
		svr.logger.Printf("main reactor exits with error:%v\n", svr.mainLoop.poller.Polling(func(fd int, ev uint32) error {
			return svr.acceptNewPriConnection(fd)
		}))

	}

}

func (svr *server) activateSubReactor(el *eventloop) {
	defer func() {
		if el.idx == 0 && svr.opts.Ticker {
			close(svr.ticktock)
		}
		svr.signalShutdown()
	}()

	if el.idx == 0 && svr.opts.Ticker {
		go el.loopTicker()
	}

	svr.logger.Printf("event-loop:%d exits with error:%v\n", el.idx, el.poller.Polling(func(fd int, ev uint32) error {
		if c, ack := el.connections[fd]; ack {
			switch c.outboundBuffer.IsEmpty() {
			// Don't change the ordering of processing EPOLLOUT | EPOLLRDHUP / EPOLLIN unless you're 100%
			// sure what you're doing!
			// Re-ordering can easily introduce bugs and bad side-effects, as I found out painfully in the past.
			case false:
				if ev&netpoll.OutEvents != 0 {
					return el.loopWrite(c)
				}
				return nil
			case true:
				if ev&netpoll.InEvents != 0 {
					return el.loopRead(c)
				}
				return nil
			}
		}
		return nil
	}))
}
