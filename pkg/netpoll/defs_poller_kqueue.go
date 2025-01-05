// Copyright (c) 2019 The Gnet Authors. All rights reserved.
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

//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package netpoll

import "golang.org/x/sys/unix"

const (
	// InitPollEventsCap represents the initial capacity of poller event-list.
	InitPollEventsCap = 64
	// MaxPollEventsCap is the maximum limitation of events that the poller can process.
	MaxPollEventsCap = 512
	// MinPollEventsCap is the minimum limitation of events that the poller can process.
	MinPollEventsCap = 16
	// MaxAsyncTasksAtOneTime is the maximum amount of asynchronous tasks that the event-loop will process at one time.
	MaxAsyncTasksAtOneTime = 128
	// ReadEvents represents readable events that are polled by kqueue.
	ReadEvents = unix.EVFILT_READ
	// WriteEvents represents writeable events that are polled by kqueue.
	WriteEvents = unix.EVFILT_WRITE
	// ReadWriteEvents represents both readable and writeable events.
	ReadWriteEvents = ReadEvents | WriteEvents
	// ErrEvents represents exceptional events that occurred.
	ErrEvents = unix.EV_EOF | unix.EV_ERROR
)

// IsReadEvent checks if the event is a read event.
func IsReadEvent(event IOEvent) bool {
	return event == ReadEvents
}

// IsWriteEvent checks if the event is a write event.
func IsWriteEvent(event IOEvent) bool {
	return event == WriteEvents
}

// IsErrorEvent checks if the event is an error event.
func IsErrorEvent(_ IOEvent, flags IOFlags) bool {
	return flags&ErrEvents != 0
}

type eventList struct {
	size   int
	events []unix.Kevent_t
}

func newEventList(size int) *eventList {
	return &eventList{size, make([]unix.Kevent_t, size)}
}

func (el *eventList) expand() {
	if newSize := el.size << 1; newSize <= MaxPollEventsCap {
		el.size = newSize
		el.events = make([]unix.Kevent_t, newSize)
	}
}

func (el *eventList) shrink() {
	if newSize := el.size >> 1; newSize >= MinPollEventsCap {
		el.size = newSize
		el.events = make([]unix.Kevent_t, newSize)
	}
}
