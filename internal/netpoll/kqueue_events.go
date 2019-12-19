// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// +build darwin netbsd freebsd openbsd dragonfly

package netpoll

import "golang.org/x/sys/unix"

const (
	// EVFilterWrite represents writeable events from sockets.
	EVFilterWrite = unix.EVFILT_WRITE
	// EVFilterRead represents readable events from sockets.
	EVFilterRead = unix.EVFILT_READ
	// EVFilterSock ...
	//EVFilterSock = -0xd
)

type eventList struct {
	size   int
	events []unix.Kevent_t
}

func newEventList(size int) *eventList {
	return &eventList{size, make([]unix.Kevent_t, size)}
}

func (el *eventList) increase() {
	el.size <<= 1
	el.events = make([]unix.Kevent_t, el.size)
}
