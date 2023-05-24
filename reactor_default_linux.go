// Copyright (c) 2019 The Gnet Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !poll_opt
// +build !poll_opt

package gnet

import (
	"runtime"

	"github.com/panjf2000/gnet/v2/internal/netpoll"
	"github.com/panjf2000/gnet/v2/pkg/errors"
)

func (el *eventloop) activateMainReactor() error {
	if el.engine.opts.LockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	err := el.poller.Polling(func(fd int, ev uint32) error { return el.engine.accept(fd, ev) })
	if err == errors.ErrEngineShutdown {
		el.engine.opts.Logger.Debugf("main reactor is exiting in terms of the demand from user, %v", err)
		err = nil
	} else if err != nil {
		el.engine.opts.Logger.Errorf("main reactor is exiting due to error: %v", err)
	}

	el.engine.shutdown(err)

	return err
}

func (el *eventloop) activateSubReactor() error {
	if el.engine.opts.LockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	err := el.poller.Polling(func(fd int, ev uint32) error {
		if c := el.connections.getConn(fd); c != nil {
			// Don't change the ordering of processing EPOLLOUT | EPOLLRDHUP / EPOLLIN unless you're 100%
			// sure what you're doing!
			// Re-ordering can easily introduce bugs and bad side-effects, as I found out painfully in the past.

			// We should always check for the EPOLLOUT event first, as we must try to send the leftover data back to
			// the peer when any error occurs on a connection.
			//
			// Either an EPOLLOUT or EPOLLERR event may be fired when a connection is refused.
			// In either case write() should take care of it properly:
			// 1) writing data back,
			// 2) closing the connection.
			if ev&netpoll.OutEvents != 0 && !c.outboundBuffer.IsEmpty() {
				if err := el.write(c); err != nil {
					return err
				}
			}
			if ev&netpoll.InEvents != 0 {
				return el.read(c)
			}
			return nil
		}
		return nil
	})

	if err == errors.ErrEngineShutdown {
		el.engine.opts.Logger.Debugf("event-loop(%d) is exiting in terms of the demand from user, %v", el.idx, err)
		err = nil
	} else if err != nil {
		el.engine.opts.Logger.Errorf("event-loop(%d) is exiting due to error: %v", el.idx, err)
	}

	el.closeConns()
	el.engine.shutdown(err)

	return err
}

func (el *eventloop) run() error {
	if el.engine.opts.LockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	err := el.poller.Polling(func(fd int, ev uint32) error {
		if c := el.connections.getConn(fd); c != nil {
			// Don't change the ordering of processing EPOLLOUT | EPOLLRDHUP / EPOLLIN unless you're 100%
			// sure what you're doing!
			// Re-ordering can easily introduce bugs and bad side-effects, as I found out painfully in the past.

			// We should always check for the EPOLLOUT event first, as we must try to send the leftover data back to
			// the peer when any error occurs on a connection.
			//
			// Either an EPOLLOUT or EPOLLERR event may be fired when a connection is refused.
			// In either case write() should take care of it properly:
			// 1) writing data back,
			// 2) closing the connection.
			if ev&netpoll.OutEvents != 0 && !c.outboundBuffer.IsEmpty() {
				if err := el.write(c); err != nil {
					return err
				}
			}
			if ev&netpoll.InEvents != 0 {
				return el.read(c)
			}
			return nil
		}
		return el.accept(fd, ev)
	})

	if err == errors.ErrEngineShutdown {
		el.engine.opts.Logger.Debugf("event-loop(%d) is exiting in terms of the demand from user, %v", el.idx, err)
		err = nil
	} else if err != nil {
		el.engine.opts.Logger.Errorf("event-loop(%d) is exiting due to error: %v", el.idx, err)
	}

	el.closeConns()
	el.engine.shutdown(err)

	return err
}
