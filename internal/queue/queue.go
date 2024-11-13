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

package queue

import "sync"

// Func is the callback function executed by poller.
type Func func(any) error

// Task is a wrapper that contains function and its argument.
type Task struct {
	Exec  Func
	Param any
}

var taskPool = sync.Pool{New: func() any { return new(Task) }}

// GetTask gets a cached Task from pool.
func GetTask() *Task {
	return taskPool.Get().(*Task)
}

// PutTask puts the trashy Task back in pool.
func PutTask(task *Task) {
	task.Exec, task.Param = nil, nil
	taskPool.Put(task)
}

// AsyncTaskQueue is a queue storing asynchronous tasks.
type AsyncTaskQueue interface {
	Enqueue(*Task)
	Dequeue() *Task
	IsEmpty() bool
	Length() int32
}

// EventPriority is the priority of an event.
type EventPriority int

const (
	// HighPriority is for the tasks expected to be executed
	// as soon as possible.
	HighPriority EventPriority = iota
	// LowPriority is for the tasks that won't matter much
	// even if they are deferred a little bit.
	LowPriority
)
