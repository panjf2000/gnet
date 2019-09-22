// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// this is a good candidate for a lock-free structure.
type spinLock uint32

func (sl *spinLock) Lock() {
	for !atomic.CompareAndSwapUint32((*uint32)(sl), 0, 1) {
		runtime.Gosched()
	}
}

func (sl *spinLock) Unlock() {
	atomic.StoreUint32((*uint32)(sl), 0)
}

type NoteQueue struct {
	mu    sync.Locker
	notes []interface{}
}

func (q *NoteQueue) Push(note interface{}) {
	q.mu.Lock()
	q.notes = append(q.notes, note)
	q.mu.Unlock()
}

func (q *NoteQueue) ForEach(iter func(note interface{}) error) error {
	q.mu.Lock()
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

func NewNoteQueue() NoteQueue {
	return NoteQueue{mu: new(spinLock)}
}
