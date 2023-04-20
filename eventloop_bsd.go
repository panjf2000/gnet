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

//go:build freebsd || dragonfly || darwin
// +build freebsd dragonfly darwin

package gnet

import (
	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/internal/netpoll"
)

func (el *eventloop) handleEvents(fd int, filter int16) (err error) {
	if gfd, ok := el.connections[fd]; ok {
		c := el.connSlice[gfd.ConnIndex1()][gfd.ConnIndex2()]
		switch filter {
		case netpoll.EVFilterSock:
			err = el.closeConn(c, unix.ECONNRESET)
		case netpoll.EVFilterWrite:
			if !c.outboundBuffer.IsEmpty() {
				err = el.write(c)
			}
		case netpoll.EVFilterRead:
			err = el.read(c)
		}
	}

	return
}
