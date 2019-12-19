// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package internal

import "sync"

// Job is a asynchronous function.
type Job func() error

// AsyncJobQueue queues pending tasks.
type AsyncJobQueue struct {
	lock sync.Locker
	jobs []func() error
}

// NewAsyncJobQueue creates a note-queue.
func NewAsyncJobQueue() AsyncJobQueue {
	return AsyncJobQueue{lock: SpinLock()}
}

// Push pushes a item into queue.
func (q *AsyncJobQueue) Push(job Job) (jobsNum int) {
	q.lock.Lock()
	q.jobs = append(q.jobs, job)
	jobsNum = len(q.jobs)
	q.lock.Unlock()
	return
}

// ForEach iterates this queue and executes each note with a given func.
func (q *AsyncJobQueue) ForEach() (err error) {
	q.lock.Lock()
	jobs := q.jobs
	q.jobs = nil
	q.lock.Unlock()
	for i := range jobs {
		if err = jobs[i](); err != nil {
			return err
		}
	}
	return
}
