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
	connAlloc  int32                      // number of allocated *conn
	fd2gfd     map[int]gfd.GFD            // fd -> gfd.GFD
}

func (cs *connStore) init() {
	cs.fd2gfd = make(map[int]gfd.GFD)
}

func (cs *connStore) iterate(f func(*conn) bool) {
	for _, conns := range cs.connMatrix {
		for _, c := range conns {
			if c != nil {
				if !f(c) {
					return
				}
			}
		}
	}
}

func (cs *connStore) incCount(i1 int, delta int32) {
	atomic.AddInt32(&cs.connCounts[i1], delta)
}

func (cs *connStore) loadCount() (n int32) {
	for i := 0; i < len(cs.connCounts); i++ {
		n += atomic.LoadInt32(&cs.connCounts[i])
	}
	return
}

func (cs *connStore) unsafeLoadCount() (n int32) {
	for i := 0; i < len(cs.connCounts); i++ {
		n += cs.connCounts[i]
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
		cs.connAlloc += gfd.ConnIndex2Max
	}
	cs.connMatrix[cs.connNAI1][cs.connNAI2] = c

	c.gfd = gfd.NewGFD(c.fd, index, cs.connNAI1, cs.connNAI2)
	cs.fd2gfd[c.fd] = c.gfd
	cs.incCount(cs.connNAI1, 1)

	for idx2 := cs.connNAI2; idx2 < gfd.ConnIndex2Max; idx2++ {
		if cs.connMatrix[cs.connNAI1][idx2] == nil {
			cs.connNAI2 = idx2
			return
		}
	}

	for idx1 := cs.connNAI1; idx1 < gfd.ConnIndex1Max; idx1++ {
		if cs.connMatrix[idx1] == nil || cs.connCounts[idx1] >= gfd.ConnIndex2Max {
			continue
		}
		for idx2 := 0; idx2 < gfd.ConnIndex2Max; idx2++ {
			if cs.connMatrix[idx1][idx2] == nil {
				cs.connNAI1, cs.connNAI2 = idx1, idx2
				return
			}
		}
	}

	for idx1 := 0; idx1 < gfd.ConnIndex1Max; idx1++ {
		if cs.connMatrix[idx1] == nil {
			cs.connNAI1, cs.connNAI2 = idx1, 0
			return
		}
	}

	cs.connNAI1 = gfd.ConnIndex1Max
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

	// Compact the connMatrix when it becomes sparse.
	const sparseFactor float32 = 0.5
	connCount := cs.unsafeLoadCount()
	if cs.connNAI1 > 0 && float32(connCount/cs.connAlloc) < sparseFactor {
		var newConnMatrix [gfd.ConnIndex1Max][]*conn
		row := connCount / gfd.ConnIndex2Max
		if connCount%gfd.ConnIndex2Max > 0 {
			row += 1
		}
		for i := int32(0); i < row; i++ {
			newConnMatrix[i] = make([]*conn, gfd.ConnIndex2Max)
			if connCount -= gfd.ConnIndex2Max; connCount >= 0 {
				atomic.StoreInt32(&cs.connCounts[i], gfd.ConnIndex2Max)
			} else {
				atomic.StoreInt32(&cs.connCounts[i], gfd.ConnIndex2Max+connCount)
			}
		}

		var i, j int
		cs.iterate(func(c *conn) bool {
			newConnMatrix[i][j] = c
			c.gfd.UpdateIndexes(i, j)
			if j++; j == gfd.ConnIndex2Max {
				i, j = i+1, 0
			}
			return true
		})

		cs.connMatrix, cs.connNAI1, cs.connNAI2 = newConnMatrix, i, j
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
