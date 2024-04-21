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

//go:build linux && !poll_opt
// +build linux,!poll_opt

package gnet

import (
	"io"
	"runtime"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/internal/netpoll"
	"github.com/panjf2000/gnet/v2/pkg/errors"
)

func (el *eventloop) rotate() error {
	if el.engine.opts.LockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	err := el.poller.Polling(el.engine.accept)
	if err == errors.ErrEngineShutdown {
		el.getLogger().Debugf("main reactor is exiting in terms of the demand from user, %v", err)
		err = nil
	} else if err != nil {
		el.getLogger().Errorf("main reactor is exiting due to error: %v", err)
	}

	el.engine.shutdown(err)

	return err
}

func (el *eventloop) orbit() error {
	if el.engine.opts.LockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	err := el.poller.Polling(func(fd int, ev uint32) error {
		c := el.connections.getConn(fd)
		if c == nil {
			// Somehow epoll notify with an event for a stale fd that is not in our connection set.
			// We need to delete it from the epoll set.
			return el.poller.Delete(fd)
		}

		// First check for any unexpected non-IO events.
		// For these events we just close the corresponding connection directly.
		if ev&netpoll.ErrEvents != 0 && ev&unix.EPOLLIN == 0 && ev&unix.EPOLLOUT == 0 {
			c.outboundBuffer.Release() // don't bother to write to a connection with some unknown error
			return el.close(c, io.EOF)
		}
		// Secondly, check for EPOLLOUT before EPOLLIN, the former has a higher priority
		// than the latter regardless of the aliveness of the current connection:
		//
		// 1. When the connection is alive and the system is overloaded, we want to
		// offload the incoming traffic by writing all pending data back to the remotes
		// before continuing to read and handle requests.
		// 2. When the connection is dead, we need to try writing any pending data back
		// to the remote and close the connection first.
		//
		// We perform eventloop.write for EPOLLOUT because it will take good care of either case.
		if ev&(unix.EPOLLOUT|unix.EPOLLERR) != 0 {
			if err := el.write(c); err != nil {
				return err
			}
		}
		// Check for EPOLLIN before EPOLLRDHUP in case that there are pending data in
		// the socket buffer.
		if ev&(unix.EPOLLIN|unix.EPOLLERR) != 0 {
			if err := el.read(c); err != nil {
				return err
			}
		}
		// Ultimately, check for EPOLLRDHUP, this event indicates that the remote has
		// either closed connection or shut down the writing half of the connection.
		if ev&unix.EPOLLRDHUP != 0 && c.opened {
			if ev&unix.EPOLLIN == 0 { // unreadable EPOLLRDHUP, close the connection directly
				return el.close(c, io.EOF)
			}
			// Received the event of EPOLLIN | EPOLLRDHUP, but the previous eventloop.read
			// failed to drain the socket buffer, so we make sure we get it done this time.
			c.isEOF = true
			return el.read(c)
		}
		return nil
	})

	if err == errors.ErrEngineShutdown {
		el.getLogger().Debugf("event-loop(%d) is exiting in terms of the demand from user, %v", el.idx, err)
		err = nil
	} else if err != nil {
		el.getLogger().Errorf("event-loop(%d) is exiting due to error: %v", el.idx, err)
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
		c := el.connections.getConn(fd)
		if c == nil {
			if _, ok := el.listeners[fd]; ok {
				return el.accept(fd, ev)
			}
			// Somehow epoll notify with an event for a stale fd that is not in our connection set.
			// We need to delete it from the epoll set.
			return el.poller.Delete(fd)

		}

		// First check for any unexpected non-IO events.
		// For these events we just close the corresponding connection directly.
		if ev&netpoll.ErrEvents != 0 && ev&unix.EPOLLIN == 0 && ev&unix.EPOLLOUT == 0 {
			c.outboundBuffer.Release() // don't bother to write to a connection with some unknown error
			return el.close(c, io.EOF)
		}
		// Secondly, check for EPOLLOUT before EPOLLIN, the former has a higher priority
		// than the latter regardless of the aliveness of the current connection:
		//
		// 1. When the connection is alive and the system is overloaded, we want to
		// offload the incoming traffic by writing all pending data back to the remotes
		// before continuing to read and handle requests.
		// 2. When the connection is dead, we need to try writing any pending data back
		// to the remote and close the connection first.
		//
		// We perform eventloop.write for EPOLLOUT because it will take good care of either case.
		if ev&(unix.EPOLLOUT|unix.EPOLLERR) != 0 {
			if err := el.write(c); err != nil {
				return err
			}
		}
		// Check for EPOLLIN before EPOLLRDHUP in case that there are pending data in
		// the socket buffer.
		if ev&(unix.EPOLLIN|unix.EPOLLERR) != 0 {
			if err := el.read(c); err != nil {
				return err
			}
		}
		// Ultimately, check for EPOLLRDHUP, this event indicates that the remote has
		// either closed connection or shut down the writing half of the connection.
		if ev&unix.EPOLLRDHUP != 0 && c.opened {
			if ev&unix.EPOLLIN == 0 { // unreadable EPOLLRDHUP, close the connection directly
				return el.close(c, io.EOF)
			}
			// Received the event of EPOLLIN | EPOLLRDHUP, but the previous eventloop.read
			// failed to drain the socket buffer, so we make sure we get it done this time.
			c.isEOF = true
			return el.read(c)
		}
		return nil
	})

	if err == errors.ErrEngineShutdown {
		el.getLogger().Debugf("event-loop(%d) is exiting in terms of the demand from user, %v", el.idx, err)
		err = nil
	} else if err != nil {
		el.getLogger().Errorf("event-loop(%d) is exiting due to error: %v", el.idx, err)
	}

	el.closeConns()
	el.engine.shutdown(err)

	return err
}
