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

package linkedlist

import (
	"io"
	"math"

	bsPool "github.com/panjf2000/gnet/v2/pkg/pool/byteslice"
)

type node struct {
	buf  []byte
	next *node
}

func (b *node) len() int {
	return len(b.buf)
}

// Buffer is a linked list of node.
type Buffer struct {
	bs    [][]byte
	head  *node
	tail  *node
	size  int
	bytes int
}

// Read reads data from the Buffer.
func (llb *Buffer) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	for b := llb.pop(); b != nil; b = llb.pop() {
		m := copy(p[n:], b.buf)
		n += m
		if m < b.len() {
			b.buf = b.buf[m:]
			llb.pushFront(b)
		} else {
			bsPool.Put(b.buf)
		}
		if n == len(p) {
			return
		}
	}
	return
}

// PushFront is a wrapper of pushFront, which accepts []byte as its argument.
func (llb *Buffer) PushFront(p []byte) {
	n := len(p)
	if n == 0 {
		return
	}
	b := bsPool.Get(n)
	copy(b, p)
	llb.pushFront(&node{buf: b})
}

// PushBack is a wrapper of pushBack, which accepts []byte as its argument.
func (llb *Buffer) PushBack(p []byte) {
	n := len(p)
	if n == 0 {
		return
	}
	b := bsPool.Get(n)
	copy(b, p)
	llb.pushBack(&node{buf: b})
}

// Peek assembles the up to maxBytes of [][]byte based on the list of node,
// it won't remove these nodes from l until Discard() is called.
func (llb *Buffer) Peek(maxBytes int) [][]byte {
	if maxBytes <= 0 {
		maxBytes = math.MaxInt32
	}
	llb.bs = llb.bs[:0]
	var cum int
	for iter := llb.head; iter != nil; iter = iter.next {
		llb.bs = append(llb.bs, iter.buf)
		if cum += iter.len(); cum >= maxBytes {
			break
		}
	}
	return llb.bs
}

// PeekWithBytes is like Peek but accepts [][]byte and puts them onto head.
func (llb *Buffer) PeekWithBytes(maxBytes int, bs ...[]byte) [][]byte {
	if maxBytes <= 0 {
		maxBytes = math.MaxInt32
	}
	llb.bs = llb.bs[:0]
	var cum int
	for _, b := range bs {
		if n := len(b); n > 0 {
			llb.bs = append(llb.bs, b)
			if cum += n; cum >= maxBytes {
				return llb.bs
			}
		}
	}
	for iter := llb.head; iter != nil; iter = iter.next {
		llb.bs = append(llb.bs, iter.buf)
		if cum += iter.len(); cum >= maxBytes {
			break
		}
	}
	return llb.bs
}

// Discard removes some nodes based on n bytes.
func (llb *Buffer) Discard(n int) (discarded int, err error) {
	if n <= 0 {
		return
	}
	for n != 0 {
		b := llb.pop()
		if b == nil {
			break
		}
		if n < b.len() {
			b.buf = b.buf[n:]
			discarded += n
			llb.pushFront(b)
			break
		}
		n -= b.len()
		discarded += b.len()
		bsPool.Put(b.buf)
	}
	return
}

const minRead = 512

// ReadFrom implements io.ReaderFrom.
func (llb *Buffer) ReadFrom(r io.Reader) (n int64, err error) {
	var m int
	for {
		b := bsPool.Get(minRead)
		m, err = r.Read(b)
		if m < 0 {
			panic("Buffer.ReadFrom: reader returned negative count from Read")
		}
		n += int64(m)
		b = b[:m]
		if err == io.EOF {
			bsPool.Put(b)
			return n, nil
		}
		if err != nil {
			bsPool.Put(b)
			return
		}
		llb.pushBack(&node{buf: b})
	}
}

// WriteTo implements io.WriterTo.
func (llb *Buffer) WriteTo(w io.Writer) (n int64, err error) {
	var m int
	for b := llb.pop(); b != nil; b = llb.pop() {
		m, err = w.Write(b.buf)
		if m > b.len() {
			panic("Buffer.WriteTo: invalid Write count")
		}
		n += int64(m)
		if err != nil {
			return
		}
		if m < b.len() {
			b.buf = b.buf[m:]
			llb.pushFront(b)
			return n, io.ErrShortWrite
		}
		bsPool.Put(b.buf)
	}
	return
}

// Len returns the length of the list.
func (llb *Buffer) Len() int {
	return llb.size
}

// Buffered returns the number of bytes that can be read from the current buffer.
func (llb *Buffer) Buffered() int {
	return llb.bytes
}

// IsEmpty reports whether l is empty.
func (llb *Buffer) IsEmpty() bool {
	return llb.head == nil
}

// Reset removes all elements from this list.
func (llb *Buffer) Reset() {
	for b := llb.pop(); b != nil; b = llb.pop() {
		bsPool.Put(b.buf)
	}
	llb.head = nil
	llb.tail = nil
	llb.size = 0
	llb.bytes = 0
	llb.bs = llb.bs[:0]
}

// pop returns and removes the head of l. If l is empty, it returns nil.
func (llb *Buffer) pop() *node {
	if llb.head == nil {
		return nil
	}
	b := llb.head
	llb.head = b.next
	if llb.head == nil {
		llb.tail = nil
	}
	b.next = nil
	llb.size--
	llb.bytes -= b.len()
	return b
}

// pushFront adds the new node to the head of l.
func (llb *Buffer) pushFront(b *node) {
	if b == nil {
		return
	}
	if llb.head == nil {
		b.next = nil
		llb.tail = b
	} else {
		b.next = llb.head
	}
	llb.head = b
	llb.size++
	llb.bytes += b.len()
}

// pushBack adds a new node to the tail of l.
func (llb *Buffer) pushBack(b *node) {
	if b == nil {
		return
	}
	if llb.tail == nil {
		llb.head = b
	} else {
		llb.tail.next = b
	}
	b.next = nil
	llb.tail = b
	llb.size++
	llb.bytes += b.len()
}
