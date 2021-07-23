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

// +build !poll_opt

package gnet

import (
	"runtime"

	"github.com/panjf2000/gnet/errors"
	"github.com/panjf2000/gnet/internal/netpoll"
)

func (el *eventloop) activateMainReactor(lockOSThread bool) {
	if lockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	defer el.svr.signalShutdown()

	err := el.poller.Polling(func(fd int, ev uint32) error { return el.svr.acceptNewConnection(ev) })
	if err == errors.ErrServerShutdown {
		el.svr.opts.Logger.Debugf("main reactor is exiting in terms of the demand from user, %v", err)
	} else if err != nil {
		el.svr.opts.Logger.Errorf("main reactor is exiting due to error: %v", err)
	}
}

func (el *eventloop) activateSubReactor(lockOSThread bool) {
	if lockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	defer func() {
		el.closeAllConns()
		el.svr.signalShutdown()
	}()

	err := el.poller.Polling(func(fd int, ev uint32) error {
		if c, ack := el.connections[fd]; ack {
			// Don't change the ordering of processing EPOLLOUT | EPOLLRDHUP / EPOLLIN unless you're 100%
			// sure what you're doing!
			// Re-ordering can easily introduce bugs and bad side-effects, as I found out painfully in the past.

			// We should always check for the EPOLLOUT event first, as we must try to send the leftover data back to
			// client when any error occurs on a connection.
			//
			// Either an EPOLLOUT or EPOLLERR event may be fired when a connection is refused.
			// In either case loopWrite() should take care of it properly:
			// 1) writing data back,
			// 2) closing the connection.
			if ev&netpoll.OutEvents != 0 && !c.outboundBuffer.IsEmpty() {
				if err := el.loopWrite(c); err != nil {
					return err
				}
			}
			// If there is pending data in outbound buffer, then we should omit this readable event
			// and prioritize the writable events to achieve a higher performance.
			//
			// Note that the client may send massive amounts of data to server by write() under blocking mode,
			// resulting in that it won't receive any responses before the server reads all data from client,
			// in which case if the server socket send buffer is full, we need to let it go and continue reading
			// the data to prevent blocking forever.
			if ev&netpoll.InEvents != 0 && (ev&netpoll.OutEvents == 0 || c.outboundBuffer.IsEmpty()) {
				return el.loopRead(c)
			}
		}
		return nil
	})
	if err == errors.ErrServerShutdown {
		el.svr.opts.Logger.Debugf("event-loop(%d) is exiting in terms of the demand from user, %v", el.idx, err)
	} else if err != nil {
		el.svr.opts.Logger.Errorf("event-loop(%d) is exiting normally on the signal error: %v", el.idx, err)
	}
}

func (el *eventloop) loopRun(lockOSThread bool) {
	if lockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	defer func() {
		el.closeAllConns()
		el.ln.close()
		el.svr.signalShutdown()
	}()

	err := el.poller.Polling(func(fd int, ev uint32) (err error) {
		if c, ok := el.connections[fd]; ok {
			// Don't change the ordering of processing EPOLLOUT | EPOLLRDHUP / EPOLLIN unless you're 100%
			// sure what you're doing!
			// Re-ordering can easily introduce bugs and bad side-effects, as I found out painfully in the past.

			// We should always check for the EPOLLOUT event first, as we must try to send the leftover data back to
			// client when any error occurs on a connection.
			//
			// Either an EPOLLOUT or EPOLLERR event may be fired when a connection is refused.
			// In either case loopWrite() should take care of it properly:
			// 1) writing data back,
			// 2) closing the connection.
			if ev&netpoll.OutEvents != 0 && !c.outboundBuffer.IsEmpty() {
				if err := el.loopWrite(c); err != nil {
					return err
				}
			}
			// If there is pending data in outbound buffer, then we should omit this readable event
			// and prioritize the writable events to achieve a higher performance.
			//
			// Note that the client may send massive amounts of data to server by write() under blocking mode,
			// resulting in that it won't receive any responses before the server read all data from client,
			// in which case if the socket send buffer is full, we need to let it go and continue reading the data
			// to prevent blocking forever.
			if ev&netpoll.InEvents != 0 && (ev&netpoll.OutEvents == 0 || c.outboundBuffer.IsEmpty()) {
				return el.loopRead(c)
			}
			return nil
		}
		return el.loopAccept(ev)
	})
	el.getLogger().Debugf("event-loop(%d) is exiting due to error: %v", el.idx, err)
}
