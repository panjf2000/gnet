// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"runtime"
	"sync/atomic"
)

// this is a good candiate for a lock-free structure.

type spinlock struct{ lock uintptr }

func (l *spinlock) Lock() {
	for !atomic.CompareAndSwapUintptr(&l.lock, 0, 1) {
		runtime.Gosched()
	}
}
func (l *spinlock) Unlock() {
	atomic.StoreUintptr(&l.lock, 0)
}

type noteQueue struct {
	mu    spinlock
	notes []interface{}
}

func (q *noteQueue) Add(note interface{}) (one bool) {
	q.mu.Lock()
	q.notes = append(q.notes, note)
	n := len(q.notes)
	q.mu.Unlock()
	return n == 1
}

func (q *noteQueue) ForEach(iter func(note interface{}) error) error {
	q.mu.Lock()
	if len(q.notes) == 0 {
		q.mu.Unlock()
		return nil
	}
	notes := q.notes
	q.notes = nil
	q.mu.Unlock()
	for _, note := range notes {
		if err := iter(note); err != nil {
			return err
		}
	}
	return nil
}
