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

//go:build (linux || freebsd || dragonfly || darwin) && gc_opt
// +build linux freebsd dragonfly darwin
// +build gc_opt

package gnet

import (
	"sync/atomic"

	"github.com/panjf2000/gnet/v2/internal/gfd"
)

type connMatrix struct {
	disableCompact bool                          // disable compaction when it is true
	connCounts     [gfd.ConnMatrixRowMax]int32   // number of active connections in event-loop
	row            int                           // next available row index
	column         int                           // next available column index
	table          [gfd.ConnMatrixRowMax][]*conn // connection matrix of *conn, multiple slices
	fd2gfd         map[int]gfd.GFD               // fd -> gfd.GFD
}

func (cm *connMatrix) init() {
	cm.fd2gfd = make(map[int]gfd.GFD)
}

func (cm *connMatrix) iterate(f func(*conn) bool) {
	cm.disableCompact = true
	defer func() { cm.disableCompact = false }()
	for _, conns := range cm.table {
		for _, c := range conns {
			if c != nil {
				if !f(c) {
					return
				}
			}
		}
	}
}

func (cm *connMatrix) incCount(row int, delta int32) {
	atomic.AddInt32(&cm.connCounts[row], delta)
}

func (cm *connMatrix) loadCount() (n int32) {
	for i := 0; i < len(cm.connCounts); i++ {
		n += atomic.LoadInt32(&cm.connCounts[i])
	}
	return
}

func (cm *connMatrix) addConn(c *conn, index int) {
	if cm.row >= gfd.ConnMatrixRowMax {
		return
	}

	if cm.table[cm.row] == nil {
		cm.table[cm.row] = make([]*conn, gfd.ConnMatrixColumnMax)
	}

	c.gfd = gfd.NewGFD(c.fd, index, cm.row, cm.column)
	cm.fd2gfd[c.fd] = c.gfd
	cm.table[cm.row][cm.column] = c
	cm.incCount(cm.row, 1)

	if cm.column++; cm.column == gfd.ConnMatrixColumnMax {
		cm.row++
		cm.column = 0
	}
}

func (cm *connMatrix) delConn(c *conn) {
	cfd, cgfd := c.fd, c.gfd

	delete(cm.fd2gfd, cfd)
	cm.incCount(cgfd.ConnMatrixRow(), -1)
	if cm.connCounts[cgfd.ConnMatrixRow()] == 0 {
		cm.table[cgfd.ConnMatrixRow()] = nil
	} else {
		cm.table[cgfd.ConnMatrixRow()][cgfd.ConnMatrixColumn()] = nil
	}
	if cm.row > cgfd.ConnMatrixRow() || cm.column > cgfd.ConnMatrixColumn() {
		cm.row, cm.column = cgfd.ConnMatrixRow(), cgfd.ConnMatrixColumn()
	}

	// Locate the last *conn in table and move it to the deleted location.

	if cm.disableCompact || cm.table[cgfd.ConnMatrixRow()] == nil { // the deleted *conn is the last one, do nothing here.
		return
	}

	// Traverse backward to find the first non-empty point in the matrix until we reach the deleted position.
	for row := gfd.ConnMatrixRowMax - 1; row >= cgfd.ConnMatrixRow(); row-- {
		if cm.connCounts[row] == 0 {
			continue
		}
		columnMin := -1
		if row == cgfd.ConnMatrixRow() {
			columnMin = cgfd.ConnMatrixColumn()
		}
		for column := gfd.ConnMatrixColumnMax - 1; column > columnMin; column-- {
			if cm.table[row][column] == nil {
				continue
			}

			gFd := cm.table[row][column].gfd
			gFd.UpdateIndexes(cgfd.ConnMatrixRow(), cgfd.ConnMatrixColumn())
			cm.table[row][column].gfd = gFd
			cm.fd2gfd[gFd.Fd()] = gFd
			cm.table[cgfd.ConnMatrixRow()][cgfd.ConnMatrixColumn()] = cm.table[row][column]
			cm.incCount(row, -1)
			cm.incCount(cgfd.ConnMatrixRow(), 1)

			if cm.connCounts[row] == 0 {
				cm.table[row] = nil
			} else {
				cm.table[row][column] = nil
			}

			cm.row, cm.column = row, column

			return
		}
	}
}

func (cm *connMatrix) getConn(fd int) *conn {
	gFD, ok := cm.fd2gfd[fd]
	if !ok {
		return nil
	}
	if cm.table[gFD.ConnMatrixRow()] == nil {
		return nil
	}
	return cm.table[gFD.ConnMatrixRow()][gFD.ConnMatrixColumn()]
}

/*
func (cm *connMatrix) getConnByGFD(fd gfd.GFD) *conn {
	if cm.table[fd.ConnMatrixRow()] == nil {
		return nil
	}
	return cm.table[fd.ConnMatrixRow()][fd.ConnMatrixColumn()]
}
*/
