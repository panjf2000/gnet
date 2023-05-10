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

package queue

import (
	"sync"

	"github.com/panjf2000/gnet/v2/pkg/gfd"
)

// TaskFunc is the callback function executed by poller.
type TaskFunc func(interface{}) error

// Task is a wrapper that contains function and its argument.
type Task struct {
	GFD      gfd.GFD
	TaskType int
	Arg      interface{}
}

var taskPool = sync.Pool{New: func() interface{} { return new(Task) }}

// GetTask gets a cached Task from pool.
func GetTask() *Task {
	return taskPool.Get().(*Task)
}

// PutTask puts the trashy Task back in pool.
func PutTask(task *Task) {
	for k := range task.GFD {
		task.GFD[k] = 0
	}
	task.TaskType, task.Arg = 0, nil
	taskPool.Put(task)
}

// AsyncTaskQueue is a queue storing asynchronous tasks.
type AsyncTaskQueue interface {
	Enqueue(*Task)
	Dequeue() *Task
	IsEmpty() bool
}
