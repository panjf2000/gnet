// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// +build darwin netbsd freebsd openbsd dragonfly

package netpoll

import "golang.org/x/sys/unix"

type eventSlice struct {
	size   int
	events []unix.Kevent_t
}

func newEventSlice(size int) *eventSlice {
	return &eventSlice{size, make([]unix.Kevent_t, size)}
}

func (es *eventSlice) increase() {
	es.size <<= 1
	es.events = make([]unix.Kevent_t, es.size)
}
