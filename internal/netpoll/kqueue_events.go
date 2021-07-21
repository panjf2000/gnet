// Copyright (c) 2019 Andy Pan
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

// +build freebsd dragonfly darwin

package netpoll

import "golang.org/x/sys/unix"

// IOEvent is the integer type of I/O events on BSD's.
type IOEvent = int16

const (
	// InitPollEventsCap represents the initial capacity of poller event-list.
	InitPollEventsCap = 64
	// MaxAsyncTasksAtOneTime is the maximum amount of asynchronous tasks that the event-loop will process at one time.
	MaxAsyncTasksAtOneTime = 128
	// EVFilterWrite represents writeable events from sockets.
	EVFilterWrite = unix.EVFILT_WRITE
	// EVFilterRead represents readable events from sockets.
	EVFilterRead = unix.EVFILT_READ
	// EVFilterSock represents exceptional events that are not read/write, like socket being closed,
	// reading/writing from/to a closed socket, etc.
	EVFilterSock = -0xd
)

type eventList struct {
	size   int
	events []unix.Kevent_t
}

func newEventList(size int) *eventList {
	return &eventList{size, make([]unix.Kevent_t, size)}
}

func (el *eventList) expand() {
	el.size <<= 1
	el.events = make([]unix.Kevent_t, el.size)
}

func (el *eventList) shrink() {
	el.size >>= 1
	el.events = make([]unix.Kevent_t, el.size)
}
