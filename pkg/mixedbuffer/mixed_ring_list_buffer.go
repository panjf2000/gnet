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
	"github.com/panjf2000/gnet/pkg/listbuffer"
	rbPool "github.com/panjf2000/gnet/pkg/pool/ringbuffer"
	"github.com/panjf2000/gnet/pkg/ringbuffer"
)

// MaxStackingBytes is the maximum amount which is allowed to be piled up in the ring-buffer.
const MaxStackingBytes = 32 * 1024 // 32KB

// Buffer combines ring-buffer and list-buffer.
// Ring-buffer is the top-priority buffer to store response data, gnet will only switch to
// list-buffer if the data size of ring-buffer reaches the maximum(MaxStackingBytes), list-buffer is more
// flexible and scalable, which helps the application reduce memory footprint.
type Buffer struct {
	ringBuffer *ringbuffer.RingBuffer
	listBuffer listbuffer.ListBuffer
}

// New instantiates a mixedbuffer.Buffer and returns it.
func New() *Buffer {
	return &Buffer{ringBuffer: rbPool.Get()}
}

// Peek returns all bytes as [][]byte, these bytes won't be discarded until Buffer.Discard() is called.
func (mb *Buffer) Peek() [][]byte {
	head, tail := mb.ringBuffer.PeekAll()
	return mb.listBuffer.PeekBytesListWithBytes(head, tail)
}

// Discard discards n bytes in this buffer.
func (mb *Buffer) Discard(n int) {
	rbLen := mb.ringBuffer.Length()
	mb.ringBuffer.Discard(n)
	if n <= rbLen {
		return
	}
	n -= rbLen
	mb.listBuffer.DiscardBytes(n)
}

// Write appends data to this buffer.
func (mb *Buffer) Write(p []byte) (int, error) {
	if !mb.listBuffer.IsEmpty() || mb.ringBuffer.Length() >= MaxStackingBytes {
		mb.listBuffer.PushBytesBack(p)
		return len(p), nil
	}
	return mb.ringBuffer.Write(p)
}

// IsEmpty indicates whether this buffer is empty.
func (mb *Buffer) IsEmpty() bool {
	return mb.ringBuffer.IsEmpty() && mb.listBuffer.IsEmpty()
}

// Release frees all resource of this buffer.
func (mb *Buffer) Release() {
	rbPool.Put(mb.ringBuffer)
	mb.listBuffer.Reset()
}
