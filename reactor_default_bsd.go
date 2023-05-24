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

//go:build (freebsd || dragonfly || darwin) && !poll_opt
// +build freebsd dragonfly darwin
// +build !poll_opt

package gnet

import (
	"runtime"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/internal/netpoll"
	"github.com/panjf2000/gnet/v2/pkg/errors"
)

func (el *eventloop) activateMainReactor() error {
	if el.engine.opts.LockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	err := el.poller.Polling(func(fd int, filter int16) error { return el.engine.accept(fd, filter) })
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

	err := el.poller.Polling(func(fd int, filter int16) (err error) {
		if c := el.connections.getConn(fd); c != nil {
			switch filter {
			case netpoll.EVFilterSock:
				err = el.close(c, unix.ECONNRESET)
			case netpoll.EVFilterWrite:
				if !c.outboundBuffer.IsEmpty() {
					err = el.write(c)
				}
			case netpoll.EVFilterRead:
				err = el.read(c)
			}
		}
		return
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

	err := el.poller.Polling(func(fd int, filter int16) (err error) {
		if c := el.connections.getConn(fd); c != nil {
			switch filter {
			case netpoll.EVFilterSock:
				err = el.close(c, unix.ECONNRESET)
			case netpoll.EVFilterWrite:
				if !c.outboundBuffer.IsEmpty() {
					err = el.write(c)
				}
			case netpoll.EVFilterRead:
				err = el.read(c)
			}
			return
		}
		return el.accept(fd, filter)
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
