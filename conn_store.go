/*
 * Copyright (c) 2023 Andy Pan, Jinxing C.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package gnet

import (
	"sync/atomic"

	"github.com/panjf2000/gnet/v2/pkg/gfd"
)

type connStore struct {
	connCounts [gfd.ConnIndex1Max]int32   // number of active connections in event-loop
	connNAI1   int                        // connections Next Available Index1
	connNAI2   int                        // connections Next Available Index2
	connMatrix [gfd.ConnIndex1Max][]*conn // connection matrix of *conn, multiple slices
	fd2gfd     map[int]gfd.GFD            // connection map: fd -> GFD
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

func (cs *connStore) addConn(i1 int, delta int32) {
	atomic.AddInt32(&cs.connCounts[i1], delta)
}

func (cs *connStore) loadConn() (n int32) {
	for i := 0; i < len(cs.connCounts); i++ {
		n += atomic.LoadInt32(&cs.connCounts[i])
	}
	return
}

// storeConn store conn
// find the next available location
// 1. current space available location
// 2. allocated other space is available
// 3. unallocated space (reapply when using it)
// 4. if no usable space is found, return directly.
func (cs *connStore) storeConn(c *conn, index int) {
	if cs.connNAI1 >= gfd.ConnIndex1Max {
		return
	}

	if cs.connMatrix[cs.connNAI1] == nil {
		cs.connMatrix[cs.connNAI1] = make([]*conn, gfd.ConnIndex2Max)
	}
	cs.connMatrix[cs.connNAI1][cs.connNAI2] = c

	c.gfd = gfd.NewGFD(c.fd, index, cs.connNAI1, cs.connNAI2)
	cs.fd2gfd[c.fd] = c.gfd
	cs.addConn(cs.connNAI1, 1)

	for i2 := cs.connNAI2; i2 < gfd.ConnIndex2Max; i2++ {
		if cs.connMatrix[cs.connNAI1][i2] == nil {
			cs.connNAI2 = i2
			return
		}
	}

	for i1 := cs.connNAI1; i1 < gfd.ConnIndex1Max; i1++ {
		if cs.connMatrix[i1] == nil || cs.connCounts[i1] >= gfd.ConnIndex2Max {
			continue
		}
		for i2 := 0; i2 < gfd.ConnIndex2Max; i2++ {
			if cs.connMatrix[i1][i2] == nil {
				cs.connNAI1, cs.connNAI2 = i1, i2
				return
			}
		}
	}

	for i1 := 0; i1 < gfd.ConnIndex1Max; i1++ {
		if cs.connMatrix[i1] == nil {
			cs.connNAI1, cs.connNAI2 = i1, 0
			return
		}
	}

	cs.connNAI1 = gfd.ConnIndex1Max
}

func (cs *connStore) removeConn(c *conn) {
	delete(cs.fd2gfd, c.fd)
	cs.addConn(c.gfd.ConnIndex1(), -1)
	if cs.connCounts[c.gfd.ConnIndex1()] == 0 {
		cs.connMatrix[c.gfd.ConnIndex1()] = nil
	} else {
		cs.connMatrix[c.gfd.ConnIndex1()][c.gfd.ConnIndex2()] = nil
	}

	if cs.connNAI1 > c.gfd.ConnIndex1() || cs.connNAI2 > c.gfd.ConnIndex2() {
		cs.connNAI1, cs.connNAI2 = c.gfd.ConnIndex1(), c.gfd.ConnIndex2()
	}
}

func (cs *connStore) getConn(fd int) *conn {
	gFD := cs.fd2gfd[fd]
	if cs.connMatrix[gFD.ConnIndex1()] == nil {
		return nil
	}
	return cs.connMatrix[gFD.ConnIndex1()][gFD.ConnIndex2()]
}

func (cs *connStore) getConnByIndex(idx1, idx2 int) *conn {
	if cs.connMatrix[idx1] == nil {
		return nil
	}
	return cs.connMatrix[idx1][idx2]
}
