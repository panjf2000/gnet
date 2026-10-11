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

//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package netpoll

import (
	"sync/atomic"

	"github.com/panjf2000/gnet/v2/pkg/queue"
)

// PollEventHandler is the callback for I/O events notified by the poller.
type PollEventHandler func(int, IOEvent, IOFlags) error

// PollAttachment is the user data which is about to be stored in "void *ptr" of epoll_data or "void *udata" of kevent.
type PollAttachment struct {
	FD       int
	Callback PollEventHandler
}

// Done returns a channel that is closed when Polling returns.
func (p *Poller) Done() <-chan struct{} {
	return p.done
}

func (p *Poller) retire() {
	if !p.stopped.CompareAndSwap(false, true) {
		return
	}
	atomic.StoreInt32(&p.wakeupCall, 0)
	p.dropQueue(p.urgentAsyncTaskQueue)
	p.dropQueue(p.asyncTaskQueue)
	close(p.done)
}

func (p *Poller) dropQueue(q queue.AsyncTaskQueue) {
	if q == nil {
		return
	}
	for task := q.Dequeue(); task != nil; task = q.Dequeue() {
		queue.PutTask(task)
	}
}
