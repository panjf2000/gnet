// Copyright (c) 2023 Andy Pan, Jinxing C.
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

//go:build (linux || freebsd || dragonfly || darwin) && gc_opt
// +build linux freebsd dragonfly darwin
// +build gc_opt

package gnet

import (
	"sync/atomic"

	"github.com/panjf2000/gnet/v2/internal/gfd"
)

type connStore struct {
	connCounts [gfd.ConnIndex1Max]int32   // number of active connections in event-loop
	connNAI1   int                        // connections Next Available Index1
	connNAI2   int                        // connections Next Available Index2
	connMatrix [gfd.ConnIndex1Max][]*conn // connection matrix of *conn, multiple slices
	fd2gfd     map[int]gfd.GFD            // fd -> gfd.GFD
}

func (cs *connStore) init() {
	cs.fd2gfd = make(map[int]gfd.GFD)
}

func (cs *connStore) iterate(f func(*conn) bool) {
	for _, conns := range cs.connMatrix {
		if len(conns) == 0 {
			continue
		}
		// Allocate new slice to keep the data safe while ranging over slice with modification in f.
		connsCopy := make([]*conn, len(conns))
		copy(connsCopy, conns)
		for _, c := range connsCopy {
			if c != nil {
				if !f(c) {
					return
				}
			}
		}
	}
}

func (cs *connStore) incCount(row int, delta int32) {
	atomic.AddInt32(&cs.connCounts[row], delta)
}

func (cs *connStore) loadCount() (n int32) {
	for i := 0; i < len(cs.connCounts); i++ {
		n += atomic.LoadInt32(&cs.connCounts[i])
	}
	return
}

// addConn stores a new conn into connStore, it finds the next available location:
// 1. current space available location
// 2. allocated other space is available
// 3. unallocated space (reapply when using it)
// 4. if no usable space is found, return directly.
func (cs *connStore) addConn(c *conn, index int) {
	if cs.connNAI1 >= gfd.ConnIndex1Max {
		return
	}

	if cs.connMatrix[cs.connNAI1] == nil {
		cs.connMatrix[cs.connNAI1] = make([]*conn, gfd.ConnIndex2Max)
	}
	cs.connMatrix[cs.connNAI1][cs.connNAI2] = c

	c.gfd = gfd.NewGFD(c.fd, index, cs.connNAI1, cs.connNAI2)
	cs.fd2gfd[c.fd] = c.gfd
	cs.incCount(cs.connNAI1, 1)

	if cs.connNAI2++; cs.connNAI2 == gfd.ConnIndex2Max {
		cs.connNAI1++
		cs.connNAI2 = 0
	}
}

func (cs *connStore) delConn(c *conn) {
	delete(cs.fd2gfd, c.fd)
	cs.incCount(c.gfd.ConnIndex1(), -1)
	if cs.connCounts[c.gfd.ConnIndex1()] == 0 {
		cs.connMatrix[c.gfd.ConnIndex1()] = nil
	} else {
		cs.connMatrix[c.gfd.ConnIndex1()][c.gfd.ConnIndex2()] = nil
	}
	if cs.connNAI1 > c.gfd.ConnIndex1() || cs.connNAI2 > c.gfd.ConnIndex2() {
		cs.connNAI1, cs.connNAI2 = c.gfd.ConnIndex1(), c.gfd.ConnIndex2()
	}

	// Locate the last *conn in connMatrix and move it to the deleted location.

	if cs.connMatrix[c.gfd.ConnIndex1()] == nil { // the deleted *conn is the last one, do nothing here.
		return
	}

	for row := gfd.ConnIndex1Max - 1; row >= c.gfd.ConnIndex1(); row-- {
		if cs.connCounts[row] == 0 {
			continue
		}
		columnMin := -1
		if row == c.gfd.ConnIndex1() {
			columnMin = c.gfd.ConnIndex2()
		}
		for column := gfd.ConnIndex2Max - 1; column > columnMin; column-- {
			if cs.connMatrix[row][column] == nil {
				continue
			}

			gFd := cs.connMatrix[row][column].gfd
			gFd.UpdateIndexes(c.gfd.ConnIndex1(), c.gfd.ConnIndex2())
			cs.connMatrix[row][column].gfd = gFd
			cs.fd2gfd[gFd.Fd()] = gFd
			cs.connMatrix[c.gfd.ConnIndex1()][c.gfd.ConnIndex2()] = cs.connMatrix[row][column]
			cs.incCount(row, -1)
			cs.incCount(c.gfd.ConnIndex1(), 1)

			if cs.connCounts[row] == 0 {
				cs.connMatrix[row] = nil
			} else {
				cs.connMatrix[row][column] = nil
			}

			cs.connNAI1, cs.connNAI2 = row, column

			return
		}
	}
}

func (cs *connStore) getConn(fd int) *conn {
	gFD, ok := cs.fd2gfd[fd]
	if !ok {
		return nil
	}
	if cs.connMatrix[gFD.ConnIndex1()] == nil {
		return nil
	}
	return cs.connMatrix[gFD.ConnIndex1()][gFD.ConnIndex2()]
}

/*
func (cs *connStore) getConnByGFD(fd gfd.GFD) *conn {
	if cs.connMatrix[fd.ConnIndex1()] == nil {
		return nil
	}
	return cs.connMatrix[fd.ConnIndex1()][fd.ConnIndex2()]
}
*/
