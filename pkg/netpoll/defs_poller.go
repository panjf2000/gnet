// Copyright (c) 2021 The Gnet Authors. All rights reserved.
// Copyright (c) 2026 Telego contributors.
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
	"errors"
	"os"
	"sync"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/pkg/queue"
)

// PollEventHandler is the callback for I/O events notified by the poller.
type PollEventHandler func(int, IOEvent, IOFlags) error

// PollAttachment is the user data which is about to be stored in "void *ptr" of epoll_data or "void *udata" of kevent.
type PollAttachment struct {
	FD       int
	Callback PollEventHandler
}

type pollerLifecycle struct {
	// Triggers share read ownership through queue insertion and notification.
	// Close takes exclusive ownership only after the polling owner has exited.
	mu       sync.RWMutex
	closed   bool
	closeErr error
}

func (p *Poller) closeDescriptors(descriptors ...int) error {
	p.lifecycle.mu.Lock()
	defer p.lifecycle.mu.Unlock()
	if p.lifecycle.closed {
		return p.lifecycle.closeErr
	}
	p.lifecycle.closed = true
	for _, descriptor := range descriptors {
		if descriptor >= 0 {
			p.lifecycle.closeErr = errors.Join(p.lifecycle.closeErr, os.NewSyscallError("close", unix.Close(descriptor)))
		}
	}
	// No producer can enqueue after this point, and the polling owner joined.
	// Drop references without executing callbacks outside their owner loop.
	for _, tasks := range []queue.AsyncTaskQueue{p.urgentAsyncTaskQueue, p.asyncTaskQueue} {
		if tasks == nil {
			continue
		}
		for task := tasks.Dequeue(); task != nil; task = tasks.Dequeue() {
			queue.PutTask(task)
		}
	}
	// A lock-free queue keeps its final dequeued node as its head. Retire the
	// queue itself so that head cannot retain a task later reused from the pool.
	p.urgentAsyncTaskQueue = nil
	p.asyncTaskQueue = nil
	return p.lifecycle.closeErr
}
