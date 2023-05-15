// Copyright (c) 2023 Jinxing C
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Package gfd provides a structure GFD to store the fd, eventloop index, connStore indexes
and some other information.

GFD structure:
|eventloop index|conn level one index|conn level two index| timestamp |      fd     |
|   1 byte      |       1 byte       |      2 byte        |  4 byte   |int type size|.
*/
package gfd

import (
	"encoding/binary"
	"math"
	"sync/atomic"
)

// Constants for GFD.
const (
	ConnIndex2Offset  = 2
	SequenceOffset    = 4
	FdOffset          = 8
	EventLoopIndexMax = math.MaxUint8 + 1
	ConnIndex1Max     = math.MaxUint8 + 1
	ConnIndex2Max     = math.MaxUint16 + 1
)

type monotoneSeq uint32

func (seq *monotoneSeq) Inc() uint32 {
	return atomic.AddUint32((*uint32)(seq), 1)
}

var monoSeq = new(monotoneSeq)

// GFD is a structure to store the fd, eventloop index, connStore indexes.
type GFD [0x10]byte

// Fd returns the underlying fd.
func (gfd GFD) Fd() int {
	return int(binary.BigEndian.Uint64(gfd[FdOffset:]))
}

// EventLoopIndex returns the eventloop index.
func (gfd GFD) EventLoopIndex() int {
	return int(gfd[0])
}

// ConnIndex1 returns the connStore index of the first level.
func (gfd GFD) ConnIndex1() int {
	return int(gfd[1])
}

// ConnIndex2 returns the connStore index of the second level.
func (gfd GFD) ConnIndex2() int {
	return int(binary.BigEndian.Uint16(gfd[ConnIndex2Offset:SequenceOffset]))
}

// Sequence returns the monotonic sequence, only used to prevent fd duplication.
func (gfd GFD) Sequence() uint32 {
	return binary.BigEndian.Uint32(gfd[SequenceOffset:FdOffset])
}

// UpdateIndexes updates the connStore indexes.
func (gfd *GFD) UpdateIndexes(idx1, idx2 int) {
	(*gfd)[1] = byte(idx1)
	binary.BigEndian.PutUint16((*gfd)[ConnIndex2Offset:SequenceOffset], uint16(idx2))
}

// NewGFD creates a new GFD.
func NewGFD(fd, elIndex, connIndex1, connIndex2 int) (gfd GFD) {
	gfd[0] = byte(elIndex)
	gfd[1] = byte(connIndex1)
	binary.BigEndian.PutUint16(gfd[ConnIndex2Offset:SequenceOffset], uint16(connIndex2))
	binary.BigEndian.PutUint32(gfd[SequenceOffset:FdOffset], monoSeq.Inc())
	binary.BigEndian.PutUint64(gfd[FdOffset:], uint64(fd))
	return
}

// Validate checks if the GFD is valid.
func Validate(gfd GFD) bool {
	return gfd.Fd() > 2 && gfd.Fd() <= math.MaxInt &&
		gfd.EventLoopIndex() >= 0 && gfd.EventLoopIndex() < EventLoopIndexMax &&
		gfd.ConnIndex1() >= 0 && gfd.ConnIndex1() < ConnIndex1Max &&
		gfd.ConnIndex2() >= 0 && gfd.ConnIndex2() < ConnIndex2Max &&
		gfd.Sequence() > 0
}
