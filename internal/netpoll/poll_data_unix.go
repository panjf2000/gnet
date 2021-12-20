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

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package netpoll

import "sync"

var pollAttachmentPool = sync.Pool{New: func() interface{} { return new(PollAttachment) }}

// GetPollAttachment attempts to get a cached PollAttachment from pool.
func GetPollAttachment() *PollAttachment {
	return pollAttachmentPool.Get().(*PollAttachment)
}

// PutPollAttachment put an unused PollAttachment back to pool.
func PutPollAttachment(pa *PollAttachment) {
	if pa == nil {
		return
	}
	pa.FD, pa.Callback = 0, nil
	pollAttachmentPool.Put(pa)
}

// PollAttachment is the user data which is about to be stored in "void *ptr" of epoll_data or "void *udata" of kevent.
type PollAttachment struct {
	FD       int
	Callback PollEventHandler
}
