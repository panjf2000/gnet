/*
 * Copyright (c) 2021 The Gnet Authors. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package gnet

import (
	"io"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/pkg/netpoll"
)

func (c *conn) processIO(_ int, ev netpoll.IOEvent, _ netpoll.IOFlags) error {
	el := c.loop
	// First check for any unexpected non-IO events.
	// For these events we just close the connection directly.
	if ev&(netpoll.ErrEvents|unix.EPOLLRDHUP) != 0 && ev&netpoll.ReadWriteEvents == 0 {
		c.outboundBuffer.Release() // don't bother to write to a connection that is already broken
		return el.close(c, io.EOF)
	}
	// Secondly, check for EPOLLOUT before EPOLLIN, the former has a higher priority
	// than the latter regardless of the aliveness of the current connection:
	//
	// 1. When the connection is alive and the system is overloaded, we want to
	// offload the incoming traffic by writing all pending data back to the remotes
	// before continuing to read and handle requests.
	// 2. When the connection is dead, we need to try writing any pending data back
	// to the remote first and then close the connection.
	//
	// We perform eventloop.write for EPOLLOUT because it can take good care of either case.
	if ev&(netpoll.WriteEvents|netpoll.ErrEvents) != 0 {
		if err := el.write(c); err != nil {
			return err
		}
	}
	// Check for EPOLLIN before EPOLLRDHUP in case that there are pending data in
	// the socket buffer.
	if ev&(netpoll.ReadEvents|netpoll.ErrEvents) != 0 {
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
		// Received the event of EPOLLIN|EPOLLRDHUP, but the previous eventloop.read
		// failed to drain the socket buffer, so we ensure to get it done this time.
		c.isEOF = true
		return el.read(c)
	}
	return nil
}
