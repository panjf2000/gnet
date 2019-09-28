// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"sync"
)

// Job ...
type Job func() error

// NewAsyncJobQueue creates a note-queue.
func NewAsyncJobQueue() AsyncJobQueue {
	return AsyncJobQueue{mu: SpinLock()}
}

// AsyncJobQueue queues pending tasks.
type AsyncJobQueue struct {
	mu   sync.Locker
	jobs []func() error
}

// Push pushes a item into queue.
func (q *AsyncJobQueue) Push(job Job) {
	q.mu.Lock()
	q.jobs = append(q.jobs, job)
	q.mu.Unlock()
}

// ForEach iterates this queue and executes each note with a given func.
func (q *AsyncJobQueue) ForEach(iter func(job Job) error) error {
	q.mu.Lock()
	jobs := q.jobs
	q.jobs = nil
	q.mu.Unlock()
	for _, job := range jobs {
		if err := iter(job); err != nil {
			return err
		}
	}
	return nil
}
