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

package listbuffer

import (
	bPool "github.com/panjf2000/gnet/pool/bytebuffer"
)

// ByteBuffer is the node of the linked list of bytes.
type ByteBuffer struct {
	Buf  *bPool.ByteBuffer
	next *ByteBuffer
}

// Len returns the length of ByteBuffer.
func (b *ByteBuffer) Len() int {
	if b.Buf == nil {
		return -1
	}
	return b.Buf.Len()
}

// IsEmpty indicates whether the ByteBuffer is empty.
func (b *ByteBuffer) IsEmpty() bool {
	if b.Buf == nil {
		return true
	}
	return b.Buf.Len() == 0
}

// ByteBufferList is a linked list of ByteBuffer.
type ByteBufferList struct {
	bs    [][]byte
	head  *ByteBuffer
	tail  *ByteBuffer
	size  int
	bytes int64
}

// Pop returns and removes the head of l. If l is empty, it returns nil.
func (l *ByteBufferList) Pop() *ByteBuffer {
	if l.head == nil {
		return nil
	}
	b := l.head
	l.head = b.next
	if l.head == nil {
		l.tail = nil
	}
	b.next = nil
	l.size--
	l.bytes -= int64(b.Buf.Len())
	return b
}

// PushFront adds the new node to the head of l.
func (l *ByteBufferList) PushFront(b *ByteBuffer) {
	if b == nil {
		return
	}
	if l.head == nil {
		b.next = nil
		l.tail = b
	} else {
		b.next = l.head
	}
	l.head = b
	l.size++
	l.bytes += int64(b.Buf.Len())
}

// PushBack adds a new node to the tail of l.
func (l *ByteBufferList) PushBack(b *ByteBuffer) {
	if b == nil {
		return
	}
	if l.tail == nil {
		l.head = b
	} else {
		l.tail.next = b
	}
	b.next = nil
	l.tail = b
	l.size++
	l.bytes += int64(b.Buf.Len())
}

// PushBytes is a wrapper of PushBack, which accepts []byte as its argument.
func (l *ByteBufferList) PushBytes(p []byte) {
	if len(p) == 0 {
		return
	}
	bb := bPool.Get()
	_, _ = bb.Write(p)
	l.PushBack(&ByteBuffer{Buf: bb})
}

// PeekBytesList assembles the [][]byte based on the list of ByteBuffer,
// it won't remove these nodes from l until DiscardBytes() is called.
func (l *ByteBufferList) PeekBytesList() [][]byte {
	l.bs = l.bs[:0]
	for iter := l.head; iter != nil; iter = iter.next {
		l.bs = append(l.bs, iter.Buf.B)
	}
	return l.bs
}

// DiscardBytes removes some nodes based on n.
func (l *ByteBufferList) DiscardBytes(n int) {
	if n <= 0 {
		return
	}
	for n != 0 {
		b := l.Pop()
		if b == nil {
			break
		}
		if n < b.Len() {
			b.Buf.B = b.Buf.B[n:]
			l.PushFront(b)
			break
		}
		n -= b.Len()
		bPool.Put(b.Buf)
	}
}

// Len returns the length of the list.
func (l *ByteBufferList) Len() int {
	return l.size
}

// Bytes returns the amount of bytes in this list.
func (l *ByteBufferList) Bytes() int64 {
	return l.bytes
}

// IsEmpty reports whether l is empty.
func (l *ByteBufferList) IsEmpty() bool {
	return l.head == nil
}

// Reset removes all elements in this list.
func (l *ByteBufferList) Reset() {
	for b := l.Pop(); b != nil; b = l.Pop() {
		bPool.Put(b.Buf)
	}
	l.head = nil
	l.tail = nil
	l.size = 0
	l.bytes = 0
	l.bs = l.bs[:0]
}
