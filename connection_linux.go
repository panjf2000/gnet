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
	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/internal/netpoll"
)

func (c *conn) handleEvents(_ int, ev uint32) error {
	if ev&netpoll.ErrEvents != 0 && ev&netpoll.InEvents == 0 && ev&netpoll.OutEvents == 0 {
		return c.loop.close(c, unix.ECONNRESET)
	}
	if ev&netpoll.OutEvents != 0 && !c.outboundBuffer.IsEmpty() {
		if err := c.loop.write(c); err != nil {
			return err
		}
	}
	if ev&netpoll.InEvents != 0 {
		return c.loop.read(c)
	}

	return nil
}

func (el *eventloop) readUDP(fd int, ev netpoll.IOEvent) error {
	return el.readUDP1(fd, ev, 0)
}
