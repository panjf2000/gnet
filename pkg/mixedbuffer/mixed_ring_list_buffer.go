// Copyright (c) 2021 Andy Pan
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

package mixedbuffer

import (
	"io"

	"github.com/panjf2000/gnet/pkg/listbuffer"
	rbPool "github.com/panjf2000/gnet/pkg/pool/ringbuffer"
	"github.com/panjf2000/gnet/pkg/ringbuffer"
)

// Buffer combines ring-buffer and list-buffer.
// Ring-buffer is the top-priority buffer to store response data, gnet will only switch to
// list-buffer if the data size of ring-buffer reaches the maximum(MaxStackingBytes), list-buffer is more
// flexible and scalable, which helps the application reduce memory footprint.
type Buffer struct {
	maxStaticBytes int
	ringBuffer     *ringbuffer.RingBuffer
	listBuffer     listbuffer.LinkedListBuffer
}

// New instantiates a mixedbuffer.Buffer and returns it.
func New(maxStaticBytes int) *Buffer {
	return &Buffer{maxStaticBytes: maxStaticBytes, ringBuffer: rbPool.Get()}
}

func (mb Buffer) Read(p []byte) (n int, err error) {
	if mb.ringBuffer == nil {
		return mb.listBuffer.Read(p)
	}
	n, err = mb.ringBuffer.Read(p)
	if n == len(p) {
		return n, err
	}
	var m int
	m, err = mb.listBuffer.Read(p[n:])
	n += m
	return
}

// Peek returns n bytes as [][]byte, these bytes won't be discarded until Buffer.Discard() is called.
func (mb *Buffer) Peek(n int) [][]byte {
	if mb.ringBuffer == nil {
		return mb.listBuffer.PeekBytesList(n)
	}

	head, tail := mb.ringBuffer.PeekAll()
	return mb.listBuffer.PeekBytesListWithBytes(n, head, tail)
}

func (mb Buffer) WriteTo(w io.Writer) (int64, error) {
	if mb.ringBuffer == nil {
		return mb.listBuffer.WriteTo(w)
	}
	return mb.ringBuffer.WriteTo(w)
}

// Discard discards n bytes in this buffer.
func (mb *Buffer) Discard(n int) {
	if mb.ringBuffer == nil {
		mb.listBuffer.Discard(n)
		return
	}

	rbLen := mb.ringBuffer.Buffered()
	mb.ringBuffer.Discard(n)
	if n <= rbLen {
		if n == rbLen {
			rbPool.Put(mb.ringBuffer)
			mb.ringBuffer = nil
		}
		return
	}
	rbPool.Put(mb.ringBuffer)
	mb.ringBuffer = nil
	n -= rbLen
	mb.listBuffer.Discard(n)
}

// Write appends data to this buffer.
func (mb *Buffer) Write(p []byte) (n int, err error) {
	if mb.ringBuffer == nil && mb.listBuffer.IsEmpty() {
		mb.ringBuffer = rbPool.Get()
	}

	if !mb.listBuffer.IsEmpty() || mb.ringBuffer.Buffered() >= mb.maxStaticBytes {
		mb.listBuffer.PushBytesBack(p)
		return len(p), nil
	}
	if mb.ringBuffer.Len() >= mb.maxStaticBytes {
		freeSize := mb.ringBuffer.Free()
		if n = len(p); n > freeSize {
			_, _ = mb.ringBuffer.Write(p[:freeSize])
			mb.listBuffer.PushBytesBack(p[freeSize:])
			return
		}
	}
	return mb.ringBuffer.Write(p)
}

func (mb *Buffer) ReadFrom(r io.Reader) (int64, error) {
	if mb.ringBuffer == nil && mb.listBuffer.IsEmpty() {
		mb.ringBuffer = rbPool.Get()
	}
	if !mb.listBuffer.IsEmpty() || mb.ringBuffer.Buffered() >= mb.maxStaticBytes {
		return mb.listBuffer.ReadFrom(r)
	}
	return mb.ringBuffer.ReadFrom(r)
}

// Writev appends multiple byte slices to this buffer.
func (mb *Buffer) Writev(bs [][]byte) (int, error) {
	if mb.ringBuffer == nil && mb.listBuffer.IsEmpty() {
		mb.ringBuffer = rbPool.Get()
	}

	if !mb.listBuffer.IsEmpty() || mb.ringBuffer.Buffered() >= mb.maxStaticBytes {
		var n int
		for _, b := range bs {
			mb.listBuffer.PushBytesBack(b)
			n += len(b)
		}
		return n, nil
	}

	var pos, cum int
	writableSize := mb.ringBuffer.Free()
	if mb.ringBuffer.Len() < mb.maxStaticBytes {
		writableSize = mb.maxStaticBytes - mb.ringBuffer.Buffered()
	}
	for i, b := range bs {
		pos = i
		cum += len(b)
		if len(b) > writableSize {
			_, _ = mb.ringBuffer.Write(b[:writableSize])
			mb.listBuffer.PushBytesBack(b[writableSize:])
			break
		}
		n, _ := mb.ringBuffer.Write(b)
		writableSize -= n
	}
	for pos++; pos < len(bs); pos++ {
		cum += len(bs[pos])
		mb.listBuffer.PushBytesBack(bs[pos])
	}
	return cum, nil
}

func (mb *Buffer) Buffered() int {
	if mb.ringBuffer == nil {
		return mb.listBuffer.Buffered()
	}
	return mb.ringBuffer.Buffered() + mb.listBuffer.Buffered()
}

// IsEmpty indicates whether this buffer is empty.
func (mb *Buffer) IsEmpty() bool {
	if mb.ringBuffer == nil {
		return mb.listBuffer.IsEmpty()
	}

	return mb.ringBuffer.IsEmpty() && mb.listBuffer.IsEmpty()
}

// Release frees all resource of this buffer.
func (mb *Buffer) Release() {
	if mb.ringBuffer != nil {
		rbPool.Put(mb.ringBuffer)
		mb.ringBuffer = nil
	}

	mb.listBuffer.Reset()
}
