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

package gfd

import (
	"encoding/binary"
	"time"
)

const (
	ConnIndex2Start = 2
	TimeStampStart  = 4
	FdStart         = 8
	ElIndexMax      = 0xFF
	ConnIndex1Max   = 0xFF
	ConnIndex2Max   = 0xFFFF
)

// GFD structure introduction.
// |eventloop index|conn level one index|conn level two index| timestamp |      fd     |
// |     1bit      |       1bit         |        2bit        |    4bit   |int type size|.
type GFD [0x10]byte

func (gfd GFD) Fd() int {
	return int(binary.BigEndian.Uint64(gfd[FdStart:]))
}

func (gfd GFD) ElIndex() int {
	return int(gfd[0])
}

func (gfd GFD) ConnIndex1() int {
	return int(gfd[1])
}

func (gfd GFD) ConnIndex2() int {
	return int(binary.BigEndian.Uint16(gfd[ConnIndex2Start:TimeStampStart]))
}

// Timestamp incomplete timestamp, only used to prevent fd duplication.
func (gfd GFD) Timestamp() uint32 {
	return binary.BigEndian.Uint32(gfd[TimeStampStart:FdStart])
}

func NewGFD(fd, elIndex, connIndex1, connIndex2 int) (gfd GFD) {
	gfd[0] = byte(elIndex)
	gfd[1] = byte(connIndex1)
	binary.BigEndian.PutUint16(gfd[ConnIndex2Start:TimeStampStart], uint16(connIndex2))
	binary.BigEndian.PutUint32(gfd[TimeStampStart:FdStart], uint32(time.Now().UnixMilli()))
	binary.BigEndian.PutUint64(gfd[FdStart:], uint64(fd))
	return
}

func CheckLegal(gfd GFD) bool {
	if gfd.Fd() <= 0 || gfd.ElIndex() < 0 || gfd.ElIndex() >= ElIndexMax ||
		gfd.ConnIndex1() < 0 || gfd.ConnIndex1() >= ConnIndex1Max ||
		gfd.ConnIndex2() < 0 || gfd.ConnIndex2() >= ConnIndex2Max ||
		gfd.Timestamp() == 0 {
		return false
	}
	return true
}
