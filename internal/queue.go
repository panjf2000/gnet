// Copyright (c) 2019 Andy Pan
// Copyright (c) 2018 Joshua J Baker
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package internal

import "sync"

// Job is a asynchronous function.
type Job func() error

// AsyncJobQueue queues pending tasks.
type AsyncJobQueue struct {
	lock sync.Locker
	jobs []Job
}

// NewAsyncJobQueue creates a note-queue.
func NewAsyncJobQueue() AsyncJobQueue {
	return AsyncJobQueue{lock: SpinLock()}
}

// Push enqueues a job onto queue.
func (q *AsyncJobQueue) Push(job Job) (pending int) {
	q.lock.Lock()
	q.jobs = append(q.jobs, job)
	pending = len(q.jobs)
	q.lock.Unlock()
	return
}

// Batch enqueues a batch of jobs onto queue.
func (q *AsyncJobQueue) Batch(jobs []Job) (pending int) {
	q.lock.Lock()
	q.jobs = append(q.jobs, jobs...)
	pending = len(q.jobs)
	q.lock.Unlock()
	return
}

// ForEach iterates this queue and executes each note with a given func.
func (q *AsyncJobQueue) ForEach() (leftover []Job, err error) {
	q.lock.Lock()
	jobs := q.jobs
	q.jobs = nil
	q.lock.Unlock()
	for i := range jobs {
		if err = jobs[i](); err != nil {
			return jobs[i+1:], err
		}
	}
	return
}
