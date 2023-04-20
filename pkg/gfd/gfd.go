package gfd

import (
	"encoding/binary"
	"time"
	"unsafe"
)

const (
	ConnIndex2Start = 2
	TimeStampStart  = 4
	FdStart         = 8
	FdSize          = unsafe.Sizeof(int(0))
	Size            = FdSize + FdStart
	ElIndexMax      = 0xFF
	ConnIndex1Max   = 0xFF
	ConnIndex2Max   = 0xFFFF
)

// GFD
// structure introduction
// |eventloop index|conn level one index|conn level two index| timestamp |      fd     |
// |     1bit      |       1bit         |        2bit        |    4bit   |int type size|
type GFD [Size]byte

func (gfd GFD) Fd() int {
	if FdSize == 4 {
		return int(binary.BigEndian.Uint32(gfd[FdStart:]))
	} else {
		return int(binary.BigEndian.Uint64(gfd[FdStart:]))
	}
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

func (gfd GFD) Timestamp() uint32 {
	return binary.BigEndian.Uint32(gfd[TimeStampStart:FdStart])
}

func NewGFD(fd, elIndex int, connIndex1, connIndex2 int) (gfd GFD) {
	gfd[0] = byte(elIndex)
	gfd[1] = byte(connIndex1)
	binary.BigEndian.PutUint16(gfd[ConnIndex2Start:TimeStampStart], uint16(connIndex2))
	binary.BigEndian.PutUint32(gfd[TimeStampStart:FdStart], uint32(time.Now().UnixMilli()))
	if FdSize == 4 {
		binary.BigEndian.PutUint32(gfd[FdStart:], uint32(fd))
	} else {
		binary.BigEndian.PutUint64(gfd[FdStart:], uint64(fd))
	}
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
