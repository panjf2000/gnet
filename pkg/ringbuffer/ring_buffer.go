// Copyright (c) 2019 Chao yuepan, Andy Pan, Allen Xu
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE

package ringbuffer

import (
	"errors"

	"github.com/panjf2000/gnet/internal/toolkit"
	"github.com/panjf2000/gnet/pkg/pool/bytebuffer"
	"github.com/panjf2000/gnet/pkg/pool/byteslice"
)

const (
	// DefaultBufferSize is the first-time allocation on a ring-buffer.
	DefaultBufferSize   = 1024     // 1KB
	bufferGrowThreshold = 4 * 1024 // 4KB
)

// MaxStreamBufferCap is the default buffer size for each stream-oriented connection(TCP/Unix).
var MaxStreamBufferCap = 64 * 1024 // 64KB

// ErrIsEmpty will be returned when trying to read an empty ring-buffer.
var ErrIsEmpty = errors.New("ring-buffer is empty")

// RingBuffer is a circular buffer that implement io.ReaderWriter interface.
type RingBuffer struct {
	bs      [][]byte
	buf     []byte
	size    int
	r       int // next position to read
	w       int // next position to write
	isEmpty bool
}

// EmptyRingBuffer can be used as a placeholder for those closed connections.
var EmptyRingBuffer = New(0)

// New returns a new RingBuffer whose buffer has the given size.
func New(size int) *RingBuffer {
	if size == 0 {
		return &RingBuffer{bs: make([][]byte, 2), isEmpty: true}
	}
	size = toolkit.CeilToPowerOfTwo(size)
	return &RingBuffer{
		bs:      make([][]byte, 2),
		buf:     make([]byte, size),
		size:    size,
		isEmpty: true,
	}
}

// Peek returns the next n bytes without advancing the read pointer.
func (rb *RingBuffer) Peek(n int) (head []byte, tail []byte) {
	if rb.isEmpty {
		return
	}

	if n <= 0 {
		return
	}

	if rb.w > rb.r {
		m := rb.w - rb.r // length of ring-buffer
		if m > n {
			m = n
		}
		head = rb.buf[rb.r : rb.r+m]
		return
	}

	m := rb.size - rb.r + rb.w // length of ring-buffer
	if m > n {
		m = n
	}

	if rb.r+m <= rb.size {
		head = rb.buf[rb.r : rb.r+m]
	} else {
		c1 := rb.size - rb.r
		head = rb.buf[rb.r:]
		c2 := m - c1
		tail = rb.buf[:c2]
	}

	return
}

// PeekAll returns all bytes without advancing the read pointer.
func (rb *RingBuffer) PeekAll() (head []byte, tail []byte) {
	if rb.isEmpty {
		return
	}

	if rb.w > rb.r {
		head = rb.buf[rb.r:rb.w]
		return
	}

	head = rb.buf[rb.r:]
	if rb.w != 0 {
		tail = rb.buf[:rb.w]
	}

	return
}

// Discard skips the next n bytes by advancing the read pointer.
func (rb *RingBuffer) Discard(n int) {
	if n <= 0 {
		return
	}

	if n < rb.Length() {
		rb.r = (rb.r + n) % rb.size
	} else {
		rb.Reset()
	}
}

// Read reads up to len(p) bytes into p. It returns the number of bytes read (0 <= n <= len(p)) and any error
// encountered.
// Even if Read returns n < len(p), it may use all of p as scratch space during the call.
// If some data is available but not len(p) bytes, Read conventionally returns what is available instead of waiting
// for more.
// When Read encounters an error or end-of-file condition after successfully reading n > 0 bytes,
// it returns the number of bytes read. It may return the (non-nil) error from the same call or return the
// error (and n == 0) from a subsequent call.
// Callers should always process the n > 0 bytes returned before considering the error err.
// Doing so correctly handles I/O errors that happen after reading some bytes and also both of the allowed EOF
// behaviors.
func (rb *RingBuffer) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	if rb.isEmpty {
		return 0, ErrIsEmpty
	}

	if rb.w > rb.r {
		n = rb.w - rb.r
		if n > len(p) {
			n = len(p)
		}
		copy(p, rb.buf[rb.r:rb.r+n])
		rb.r += n
		if rb.r == rb.w {
			rb.Reset()
		}
		return
	}

	n = rb.size - rb.r + rb.w
	if n > len(p) {
		n = len(p)
	}

	if rb.r+n <= rb.size {
		copy(p, rb.buf[rb.r:rb.r+n])
	} else {
		c1 := rb.size - rb.r
		copy(p, rb.buf[rb.r:])
		c2 := n - c1
		copy(p[c1:], rb.buf[:c2])
	}
	rb.r = (rb.r + n) % rb.size
	if rb.r == rb.w {
		rb.Reset()
	}

	return
}

// ReadByte reads and returns the next byte from the input or ErrIsEmpty.
func (rb *RingBuffer) ReadByte() (b byte, err error) {
	if rb.isEmpty {
		return 0, ErrIsEmpty
	}
	b = rb.buf[rb.r]
	rb.r++
	if rb.r == rb.size {
		rb.r = 0
	}
	if rb.r == rb.w {
		rb.Reset()
	}

	return
}

// Write writes len(p) bytes from p to the underlying buf.
// It returns the number of bytes written from p (n == len(p) > 0) and any error encountered that caused the write to
// stop early.
// If the length of p is greater than the writable capacity of this ring-buffer, it will allocate more memory to
// this ring-buffer.
// Write must not modify the slice data, even temporarily.
func (rb *RingBuffer) Write(p []byte) (n int, err error) {
	n = len(p)
	if n == 0 {
		return
	}

	free := rb.Free()
	if n > free {
		rb.grow(rb.size + n - free)
	}

	if rb.w >= rb.r {
		c1 := rb.size - rb.w
		if c1 >= n {
			copy(rb.buf[rb.w:], p)
			rb.w += n
		} else {
			copy(rb.buf[rb.w:], p[:c1])
			c2 := n - c1
			copy(rb.buf, p[c1:])
			rb.w = c2
		}
	} else {
		copy(rb.buf[rb.w:], p)
		rb.w += n
	}

	if rb.w == rb.size {
		rb.w = 0
	}

	rb.isEmpty = false

	return
}

// WriteByte writes one byte into buffer.
func (rb *RingBuffer) WriteByte(c byte) error {
	if rb.Free() < 1 {
		rb.grow(1)
	}
	rb.buf[rb.w] = c
	rb.w++

	if rb.w == rb.size {
		rb.w = 0
	}
	rb.isEmpty = false

	return nil
}

// Length returns the length of available bytes to read.
func (rb *RingBuffer) Length() int {
	if rb.r == rb.w {
		if rb.isEmpty {
			return 0
		}
		return rb.size
	}

	if rb.w > rb.r {
		return rb.w - rb.r
	}

	return rb.size - rb.r + rb.w
}

// Len returns the length of the underlying buffer.
func (rb *RingBuffer) Len() int {
	return len(rb.buf)
}

// Cap returns the size of the underlying buffer.
func (rb *RingBuffer) Cap() int {
	return rb.size
}

// Free returns the length of available bytes to write.
func (rb *RingBuffer) Free() int {
	if rb.r == rb.w {
		if rb.isEmpty {
			return rb.size
		}
		return 0
	}

	if rb.w < rb.r {
		return rb.r - rb.w
	}

	return rb.size - rb.w + rb.r
}

// WriteString writes the contents of the string s to buffer, which accepts a slice of bytes.
func (rb *RingBuffer) WriteString(s string) (int, error) {
	return rb.Write(toolkit.StringToBytes(s))
}

// ByteBuffer returns all available read bytes. It does not move the read pointer and only copy the available data.
func (rb *RingBuffer) ByteBuffer() *bytebuffer.ByteBuffer {
	if rb.isEmpty {
		return nil
	} else if rb.w == rb.r {
		bb := bytebuffer.Get()
		_, _ = bb.Write(rb.buf[rb.r:])
		_, _ = bb.Write(rb.buf[:rb.w])
		return bb
	}

	bb := bytebuffer.Get()
	if rb.w > rb.r {
		_, _ = bb.Write(rb.buf[rb.r:rb.w])
		return bb
	}

	_, _ = bb.Write(rb.buf[rb.r:])

	if rb.w != 0 {
		_, _ = bb.Write(rb.buf[:rb.w])
	}

	return bb
}

// WithByteBuffer combines the available read bytes and the given bytes. It does not move the read pointer and
// only copy the available data.
func (rb *RingBuffer) WithByteBuffer(b []byte) *bytebuffer.ByteBuffer {
	if rb.isEmpty {
		return &bytebuffer.ByteBuffer{B: b}
	} else if rb.w == rb.r {
		bb := bytebuffer.Get()
		_, _ = bb.Write(rb.buf[rb.r:])
		_, _ = bb.Write(rb.buf[:rb.w])
		_, _ = bb.Write(b)
		return bb
	}

	bb := bytebuffer.Get()
	if rb.w > rb.r {
		_, _ = bb.Write(rb.buf[rb.r:rb.w])
		_, _ = bb.Write(b)
		return bb
	}

	_, _ = bb.Write(rb.buf[rb.r:])

	if rb.w != 0 {
		_, _ = bb.Write(rb.buf[:rb.w])
	}
	_, _ = bb.Write(b)

	return bb
}

// IsFull tells if this ring-buffer is full.
func (rb *RingBuffer) IsFull() bool {
	return rb.r == rb.w && !rb.isEmpty
}

// IsEmpty tells if this ring-buffer is empty.
func (rb *RingBuffer) IsEmpty() bool {
	return rb.isEmpty
}

// Reset the read pointer and write pointer to zero.
func (rb *RingBuffer) Reset() {
	rb.isEmpty = true
	rb.r, rb.w = 0, 0
}

func (rb *RingBuffer) grow(newCap int) {
	if n := rb.size; n == 0 {
		if newCap <= DefaultBufferSize {
			newCap = DefaultBufferSize
		} else {
			newCap = toolkit.CeilToPowerOfTwo(newCap)
		}
	} else {
		doubleCap := n + n
		if newCap <= doubleCap {
			if n < bufferGrowThreshold {
				newCap = doubleCap
			} else {
				// Check 0 < n to detect overflow and prevent an infinite loop.
				for 0 < n && n < newCap {
					n += n / 4
				}
				// The n calculation doesn't overflow, set n to newCap.
				if n > 0 {
					newCap = n
				}
			}
		}
	}
	newBuf := byteslice.Get(newCap)
	oldLen := rb.Length()
	_, _ = rb.Read(newBuf)
	byteslice.Put(rb.buf)
	rb.buf = newBuf
	rb.r = 0
	rb.w = oldLen
	rb.size = newCap
	if rb.w > 0 {
		rb.isEmpty = false
	}
}
