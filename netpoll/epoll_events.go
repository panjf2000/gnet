// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build linux

package netpoll

import "golang.org/x/sys/unix"

type eventSlice struct {
	size   int
	events []unix.EpollEvent
}

func newEventSlice(size int) *eventSlice {
	return &eventSlice{size, make([]unix.EpollEvent, size)}
}

func (es *eventSlice) increase() {
	es.size <<= 1
	es.events = make([]unix.EpollEvent, es.size)
}
