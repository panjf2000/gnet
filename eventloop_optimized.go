// Copyright (c) 2023 Jinxing C
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

//go:build poll_opt
// +build poll_opt

package gnet

import "github.com/panjf2000/gnet/v2/internal/netpoll"

func (el *eventloop) pollCallback(poolType netpoll.PollCallbackType, fd int, e netpoll.IOEvent) (err error) {
	switch poolType {
	case netpoll.PollAttachmentMainAccept:
		return el.engine.accept(fd, e)
	case netpoll.PollAttachmentEventLoops:
		return el.accept(fd, e)
	case netpoll.PollAttachmentStream:
		return el.handleEvents(fd, e)
	case netpoll.PollAttachmentDatagram:
		return el.readUDP(fd, e)
	default:
		return
	}
}
