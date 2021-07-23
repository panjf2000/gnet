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

// +build freebsd dragonfly darwin
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

	err := el.poller.Polling(func(fd int, filter int16) error { return el.svr.acceptNewConnection(filter) })
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

	err := el.poller.Polling(func(fd int, filter int16) (err error) {
		if c, ack := el.connections[fd]; ack {
			switch filter {
			case netpoll.EVFilterSock:
				err = el.loopCloseConn(c, nil)
			case netpoll.EVFilterWrite:
				if !c.outboundBuffer.IsEmpty() {
					err = el.loopWrite(c)
				}
			case netpoll.EVFilterRead:
				err = el.loopRead(c)
			}
		}
		return
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

	err := el.poller.Polling(func(fd int, filter int16) (err error) {
		if c, ack := el.connections[fd]; ack {
			switch filter {
			case netpoll.EVFilterSock:
				err = el.loopCloseConn(c, nil)
			case netpoll.EVFilterWrite:
				if !c.outboundBuffer.IsEmpty() {
					err = el.loopWrite(c)
				}
			case netpoll.EVFilterRead:
				err = el.loopRead(c)
			}
			return
		}
		return el.loopAccept(filter)
	})
	el.getLogger().Debugf("event-loop(%d) is exiting due to error: %v", el.idx, err)
}
