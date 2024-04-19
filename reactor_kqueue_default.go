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

//go:build (freebsd || dragonfly || netbsd || openbsd || darwin) && !poll_opt
// +build freebsd dragonfly netbsd openbsd darwin
// +build !poll_opt

package gnet

import (
	"io"
	"runtime"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/pkg/errors"
)

func (el *eventloop) launch() error {
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

	err := el.poller.Polling(func(fd int, filter int16, flags uint16) (err error) {
		c := el.connections.getConn(fd)
		if c == nil {
			// This could happen when the connection has already been closed,
			// the file descriptor will be deleted from kqueue automatically
			// as documented in the manual pages, So we just print a warning log.
			el.getLogger().Warnf("event[fd=%d|filter=%d|flags=%d] of a stale connection from event-loop(%d)", fd, filter, flags, el.idx)
			return
		}

		switch filter {
		case unix.EVFILT_READ:
			err = el.read(c)
		case unix.EVFILT_WRITE:
			err = el.write(c)
		}
		// EV_EOF indicates that the peer has closed the connection.
		// We check for EV_EOF after processing the read/write event
		// to ensure that nothing is left out on this event filter.
		if flags&unix.EV_EOF != 0 && c.opened {
			switch filter {
			case unix.EVFILT_READ:
				// Receive the event of EVFILT_READ | EV_EOF, but the previous eventloop.read
				// failed to drain the socket buffer, so we make sure we get it done this time.
				c.isEOF = true
				err = el.read(c)
			case unix.EVFILT_WRITE:
				// The peer is disconnected, don't bother to try writing pending data back.
				c.outboundBuffer.Release()
				fallthrough
			default:
				err = el.close(c, io.EOF)
			}
		}
		return
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

	err := el.poller.Polling(func(fd int, filter int16, flags uint16) (err error) {
		c := el.connections.getConn(fd)
		if c == nil {
			if fd == el.ln.fd {
				return el.accept(fd, filter, flags)
			}
			// This could happen when the connection has already been closed,
			// the file descriptor will be deleted from kqueue automatically
			// as documented in the manual pages, So we just print a warning log.
			el.getLogger().Warnf("event[fd=%d|filter=%d|flags=%d] of a stale connection from event-loop(%d)", fd, filter, flags, el.idx)
			return
		}

		switch filter {
		case unix.EVFILT_READ:
			err = el.read(c)
		case unix.EVFILT_WRITE:
			err = el.write(c)
		}
		// EV_EOF indicates that the peer has closed the connection.
		// We check for EV_EOF after processing the read/write event
		// to ensure that nothing is left out on this event filter.
		if flags&unix.EV_EOF != 0 && c.opened {
			switch filter {
			case unix.EVFILT_READ:
				// Receive the event of EVFILT_READ | EV_EOF, but the previous eventloop.read
				// failed to drain the socket buffer, so we make sure we get it done this time.
				c.isEOF = true
				err = el.read(c)
			case unix.EVFILT_WRITE:
				// The peer is disconnected, don't bother to try writing pending data back.
				c.outboundBuffer.Release()
				fallthrough
			default:
				err = el.close(c, io.EOF)
			}
		}
		return
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
