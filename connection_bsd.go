// Copyright (c) 2021 The Gnet Authors. All rights reserved.
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

//go:build freebsd || dragonfly || netbsd || openbsd || darwin
// +build freebsd dragonfly netbsd openbsd darwin

package gnet

import (
	"io"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/internal/netpoll"
)

func (c *conn) handleEvents(_ int, filter int16, flags uint16) (err error) {
	el := c.loop
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
}

func (el *eventloop) readUDP(fd int, filter netpoll.IOEvent, flags netpoll.IOFlags) error {
	return el.readUDP1(fd, filter, flags)
}
