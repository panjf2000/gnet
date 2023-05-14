// Copyright (c) 2023 Andy Pan.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build (linux || freebsd || dragonfly || darwin) && !gc_opt
// +build linux freebsd dragonfly darwin
// +build !gc_opt

package gnet

import (
	"sync/atomic"

	"github.com/panjf2000/gnet/v2/pkg/gfd"
)

type connStore struct {
	connCount int32
	connMap   map[int]*conn
}

func (cs *connStore) init() {
	cs.connMap = make(map[int]*conn)
}

func (cs *connStore) iterate(f func(*conn) bool) {
	for _, c := range cs.connMap {
		if c != nil {
			if !f(c) {
				return
			}
		}
	}
}

func (cs *connStore) incCount(_ int, delta int32) {
	atomic.AddInt32(&cs.connCount, delta)
}

func (cs *connStore) loadCount() (n int32) {
	return atomic.LoadInt32(&cs.connCount)
}

func (cs *connStore) addConn(c *conn, index int) {
	c.gfd = gfd.NewGFD(c.fd, index, 0, 0)
	cs.connMap[c.fd] = c
}

func (cs *connStore) delConn(c *conn) {
	delete(cs.connMap, c.fd)
	cs.incCount(0, -1)
}

func (cs *connStore) getConn(fd int) *conn {
	return cs.connMap[fd]
}

func (cs *connStore) getConnByGFD(fd gfd.GFD) *conn {
	return cs.connMap[fd.Fd()]
}
