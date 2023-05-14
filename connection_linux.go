// Copyright (c) 2021 Andy Pan
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

//go:build linux
// +build linux

package gnet

import "github.com/panjf2000/gnet/v2/internal/netpoll"

func (c *conn) handleEvents(_ int, ev uint32) error {
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
		if err := c.loop.write(c); err != nil {
			return err
		}
	}
	if ev&netpoll.InEvents != 0 {
		return c.loop.read(c)
	}

	return nil
}
