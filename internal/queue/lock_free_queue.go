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

// Package queue delivers an implementation of lock-free concurrent queue based on
// the algorithm presented by Maged M. Michael and Michael L. Scot. in 1996: https://dl.acm.org/doi/10.1145/248052.248106
//
// Pseudocode of non-blocking concurrent queue algorithm:
/*
	structure pointer_t {ptr: pointer to node_t, count: unsigned integer}
	structure node_t {value: data type, next: pointer_t}
	structure queue_t {Head: pointer_t, Tail: pointer_t}

	initialize(Q: pointer to queue_t)
	node = new_node()		// Allocate a free node
	node->next.ptr = NULL	// Make it the only node in the linked list
	Q->Head.ptr = Q->Tail.ptr = node	// Both Head and Tail point to it

	enqueue(Q: pointer to queue_t, value: data type)
	E1:   node = new_node()	// Allocate a new node from the free list
	E2:   node->value = value	// Copy enqueued value into node
	E3:   node->next.ptr = NULL	// Set next pointer of node to NULL
	E4:   loop			// Keep trying until Enqueue is done
	E5:      tail = Q->Tail	// Read Tail.ptr and Tail.count together
	E6:      next = tail.ptr->next	// Read next ptr and count fields together
	E7:      if tail == Q->Tail	// Are tail and next consistent?
				// Was Tail pointing to the last node?
	E8:         if next.ptr == NULL
					// Try to link node at the end of the linked list
	E9:            if CAS(&tail.ptr->next, next, <node, next.count+1>)
	E10:               break	// Enqueue is done.  Exit loop
	E11:            endif
	E12:         else		// Tail was not pointing to the last node
					// Try to swing Tail to the next node
	E13:            CAS(&Q->Tail, tail, <next.ptr, tail.count+1>)
	E14:         endif
	E15:      endif
	E16:   endloop
			// Enqueue is done.  Try to swing Tail to the inserted node
	E17:   CAS(&Q->Tail, tail, <node, tail.count+1>)

	dequeue(Q: pointer to queue_t, pvalue: pointer to data type): boolean
	D1:   loop			     // Keep trying until Dequeue is done
	D2:      head = Q->Head	     // Read Head
	D3:      tail = Q->Tail	     // Read Tail
	D4:      next = head.ptr->next    // Read Head.ptr->next
	D5:      if head == Q->Head	     // Are head, tail, and next consistent?
	D6:         if head.ptr == tail.ptr // Is queue empty or Tail falling behind?
	D7:            if next.ptr == NULL  // Is queue empty?
	D8:               return FALSE      // Queue is empty, couldn't dequeue
	D9:            endif
					// Tail is falling behind.  Try to advance it
	D10:            CAS(&Q->Tail, tail, <next.ptr, tail.count+1>)
	D11:         else		     // No need to deal with Tail
					// Read value before CAS
					// Otherwise, another dequeue might free the next node
	D12:            *pvalue = next.ptr->value
					// Try to swing Head to the next node
	D13:            if CAS(&Q->Head, head, <next.ptr, head.count+1>)
	D14:               break             // Dequeue is done.  Exit loop
	D15:            endif
	D16:         endif
	D17:      endif
	D18:   endloop
	D19:   free(head.ptr)		     // It is safe now to free the old node
	D20:   return TRUE                   // Queue was not empty, dequeue succeeded
*/
package queue

import (
	"sync/atomic"
	"unsafe"
)

// lockFreeQueue is a simple, fast, and practical non-blocking and concurrent queue with no lock.
type lockFreeQueue struct {
	head   unsafe.Pointer
	tail   unsafe.Pointer
	length int32
}

type node struct {
	value *Task
	next  unsafe.Pointer
}

// NewLockFreeQueue instantiates and returns a lockFreeQueue.
func NewLockFreeQueue() AsyncTaskQueue {
	n := unsafe.Pointer(&node{})
	return &lockFreeQueue{head: n, tail: n}
}

// Enqueue puts the given value v at the tail of the queue.
func (q *lockFreeQueue) Enqueue(task *Task) {
	n := &node{value: task}
retry:
	tail := load(&q.tail)
	next := load(&tail.next)
	// Are tail and next consistent?
	if tail == load(&q.tail) {
		if next == nil {
			// Try to link node at the end of the linked list.
			if cas(&tail.next, next, n) { // enqueue is done.
				// Try to swing tail to the inserted node.
				cas(&q.tail, tail, n)
				atomic.AddInt32(&q.length, 1)
				return
			}
		} else { // tail was not pointing to the last node
			// Try to swing tail to the next node.
			cas(&q.tail, tail, next)
		}
	}
	goto retry
}

// Dequeue removes and returns the value at the head of the queue.
// It returns nil if the queue is empty.
func (q *lockFreeQueue) Dequeue() *Task {
retry:
	head := load(&q.head)
	tail := load(&q.tail)
	next := load(&head.next)
	// Are head, tail, and next consistent?
	if head == load(&q.head) {
		// Is queue empty or tail falling behind?
		if head == tail {
			// Is queue empty?
			if next == nil {
				return nil
			}
			cas(&q.tail, tail, next) // tail is falling behind, try to advance it.
		} else {
			// Read value before CAS, otherwise another dequeue might free the next node.
			task := next.value
			if cas(&q.head, head, next) { // dequeue is done, return value.
				atomic.AddInt32(&q.length, -1)
				return task
			}
		}
	}
	goto retry
}

// IsEmpty indicates whether this queue is empty or not.
func (q *lockFreeQueue) IsEmpty() bool {
	return atomic.LoadInt32(&q.length) == 0
}

func load(p *unsafe.Pointer) (n *node) {
	return (*node)(atomic.LoadPointer(p))
}

func cas(p *unsafe.Pointer, old, new *node) bool {
	return atomic.CompareAndSwapPointer(p, unsafe.Pointer(old), unsafe.Pointer(new))
}
