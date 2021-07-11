// Copyright (c) 2021 Andy Pan
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

package queue

import "sync"

// TaskFunc is the callback function executed by poller.
type TaskFunc func(interface{}) error

// Task is a wrapper that contains function and its argument.
type Task struct {
	Run TaskFunc
	Arg interface{}
}

var taskPool = sync.Pool{New: func() interface{} { return new(Task) }}

// GetTask gets a cached Task from pool.
func GetTask() *Task {
	return taskPool.Get().(*Task)
}

// PutTask puts the trashy Task back in pool.
func PutTask(task *Task) {
	task.Run, task.Arg = nil, nil
	taskPool.Put(task)
}

// AsyncTaskQueue is a queue storing asynchronous tasks.
type AsyncTaskQueue interface {
	Enqueue(*Task)
	Dequeue() *Task
	Empty() bool
}
