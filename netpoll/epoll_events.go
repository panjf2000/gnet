// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build linux

package netpoll

import "golang.org/x/sys/unix"

const (
	// ErrEvents ...
	ErrEvents = unix.EPOLLERR | unix.EPOLLHUP
	// OutEvents ...
	OutEvents = unix.EPOLLOUT
	// InEvents ...
	InEvents = ErrEvents | unix.EPOLLRDHUP | unix.EPOLLIN
)

type eventList struct {
	size   int
	events []unix.EpollEvent
}

func newEventList(size int) *eventList {
	return &eventList{size, make([]unix.EpollEvent, size)}
}

func (el *eventList) increase() {
	el.size <<= 1
	el.events = make([]unix.EpollEvent, el.size)
}
