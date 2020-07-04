// Copyright (c) 2019 Andy Pan
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package gnet

import (
	"github.com/panjf2000/gnet/errors"
	"github.com/panjf2000/gnet/internal/netpoll"
)

func (svr *server) activateMainReactor() {
	defer svr.signalShutdown()
	switch err := svr.mainLoop.poller.Polling(func(fd int, ev uint32) error { return svr.acceptNewConnection(fd) }); err {
	case errors.ErrServerShutdown:
		svr.logger.Infof("Main reactor is exiting normally on the signal error: %v", err)
	default:
		svr.logger.Errorf("Main reactor is exiting due to an unexpected error: %v", err)
	}
}

func (svr *server) activateSubReactor(el *eventloop) {
	defer func() {
		el.closeAllConns()
		if el.idx == 0 && svr.opts.Ticker {
			close(svr.ticktock)
		}
		svr.signalShutdown()
	}()

	if el.idx == 0 && svr.opts.Ticker {
		go el.loopTicker()
	}
	switch err := el.poller.Polling(func(fd int, ev uint32) error {
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
	}); err {
	case errors.ErrServerShutdown:
		svr.logger.Infof("Event-loop(%d) is exiting normally on the signal error: %v", el.idx, err)
	default:
		svr.logger.Errorf("Event-loop(%d) is exiting due to an unexpected error: %v", el.idx, err)
	}
}
