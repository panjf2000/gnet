// Copyright (c) 2023 The Gnet Authors. All rights reserved.
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

	"github.com/panjf2000/gnet/v2/internal/gfd"
)

type connMatrix struct {
	connCount int32
	connMap   map[int]*conn
}

func (cm *connMatrix) init() {
	cm.connMap = make(map[int]*conn)
}

func (cm *connMatrix) iterate(f func(*conn) bool) {
	for _, c := range cm.connMap {
		if c != nil {
			if !f(c) {
				return
			}
		}
	}
}

func (cm *connMatrix) incCount(_ int, delta int32) {
	atomic.AddInt32(&cm.connCount, delta)
}

func (cm *connMatrix) loadCount() (n int32) {
	return atomic.LoadInt32(&cm.connCount)
}

func (cm *connMatrix) addConn(c *conn, index int) {
	c.gfd = gfd.NewGFD(c.fd, index, 0, 0)
	cm.connMap[c.fd] = c
	cm.incCount(0, 1)
}

func (cm *connMatrix) delConn(c *conn) {
	delete(cm.connMap, c.fd)
	cm.incCount(0, -1)
}

func (cm *connMatrix) getConn(fd int) *conn {
	return cm.connMap[fd]
}

/*
func (cm *connMatrix) getConnByGFD(fd gfd.GFD) *conn {
	return cm.connMap[fd.Fd()]
}
*/
