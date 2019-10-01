// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// +build darwin netbsd freebsd openbsd dragonfly

package netpoll

import "golang.org/x/sys/unix"

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
