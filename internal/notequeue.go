// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"sync"
)

// NewNoteQueue creates a note-queue.
func NewNoteQueue() NoteQueue {
	return NoteQueue{mu: SpinLock()}
}

// NoteQueue queues pending tasks.
type NoteQueue struct {
	mu    sync.Locker
	notes []interface{}
}

// Push pushes a item into queue.
func (q *NoteQueue) Push(note interface{}) {
	q.mu.Lock()
	q.notes = append(q.notes, note)
	q.mu.Unlock()
}

// ForEach iterates this queue and executes each note with a given func.
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
