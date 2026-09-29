// Copyright (c) 2026 The Gnet Authors. All rights reserved.
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

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/queue"
)

// Done is closed when Polling returns. The close happens after queued tasks
// have been dropped, so a waiter that observes Done and has not observed its
// own completion signal still owns whatever resource the dropped task held.
func (p *Poller) Done() <-chan struct{} { return p.done }

// enqueue admits a task unless the poller has retired. The stopped check and
// the enqueue form one critical section under the read lock, so retire, which
// takes the write lock, drains only after every admitted task is in a queue.
// A task accepted here is therefore either executed by Polling or dropped by
// retire before Done is closed; none can be left behind the final drain.
func (p *Poller) enqueue(priority queue.EventPriority, fn queue.Func, param any) error {
	p.retireMu.RLock()
	defer p.retireMu.RUnlock()
	if p.stopped {
		return errorx.ErrEngineShutdown
	}
	task := queue.GetTask()
	task.Exec, task.Param = fn, param
	if priority > queue.HighPriority && p.urgentAsyncTaskQueue.Length() >= p.highPriorityEventsThreshold {
		p.asyncTaskQueue.Enqueue(task)
	} else {
		// There might be some low-priority tasks overflowing into urgentAsyncTaskQueue in a flash,
		// but that's tolerable because it ought to be a rare case.
		p.urgentAsyncTaskQueue.Enqueue(task)
	}
	return nil
}

// retire runs on every return from Polling. Taking the write lock waits for
// every Trigger already past the stopped check, and setting stopped under it
// rejects every later one, so the drain below is final.
func (p *Poller) retire() {
	p.retireMu.Lock()
	defer p.retireMu.Unlock()
	if p.stopped {
		return
	}
	p.stopped = true
	atomic.StoreInt32(&p.wakeupCall, 0)
	p.dropQueue(p.urgentAsyncTaskQueue)
	p.dropQueue(p.asyncTaskQueue)
	close(p.done)
}

// dropQueue discards tasks without executing them. register would attach a
// connection to a poller that is about to be closed; waiters learn of the drop
// from Done instead.
func (p *Poller) dropQueue(q queue.AsyncTaskQueue) {
	if q == nil {
		return
	}
	for task := q.Dequeue(); task != nil; task = q.Dequeue() {
		queue.PutTask(task)
	}
}
