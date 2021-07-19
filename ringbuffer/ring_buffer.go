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

	"github.com/panjf2000/gnet/internal"
	"github.com/panjf2000/gnet/pool/bytebuffer"
)

const (
	defaultBufferSize   = 1024     // 1KB for the first-time allocation on ring-buffer.
	bufferGrowThreshold = 4 * 1024 // 4KB
)

// ErrIsEmpty will be returned when trying to read a empty ring-buffer.
var ErrIsEmpty = errors.New("ring-buffer is empty")

// RingBuffer is a circular buffer that implement io.ReaderWriter interface.
type RingBuffer struct {
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
		return &RingBuffer{isEmpty: true}
	}
	size = internal.CeilToPowerOfTwo(size)
	return &RingBuffer{
		buf:     make([]byte, size),
		size:    size,
		isEmpty: true,
	}
}

// Peek returns the next n bytes without advancing the read pointer.
func (r *RingBuffer) Peek(n int) (head []byte, tail []byte) {
	if r.isEmpty {
		return
	}

	if n <= 0 {
		return
	}

	if r.w > r.r {
		m := r.w - r.r // length of ring-buffer
		if m > n {
			m = n
		}
		head = r.buf[r.r : r.r+m]
		return
	}

	m := r.size - r.r + r.w // length of ring-buffer
	if m > n {
		m = n
	}

	if r.r+m <= r.size {
		head = r.buf[r.r : r.r+m]
	} else {
		c1 := r.size - r.r
		head = r.buf[r.r:]
		c2 := m - c1
		tail = r.buf[:c2]
	}

	return
}

// PeekAll returns all bytes without advancing the read pointer.
func (r *RingBuffer) PeekAll() (head []byte, tail []byte) {
	if r.isEmpty {
		return
	}

	if r.w > r.r {
		head = r.buf[r.r:r.w]
		return
	}

	head = r.buf[r.r:]
	if r.w != 0 {
		tail = r.buf[:r.w]
	}

	return
}

// Discard skips the next n bytes by advancing the read pointer.
func (r *RingBuffer) Discard(n int) {
	if n <= 0 {
		return
	}

	if n < r.Length() {
		r.r = (r.r + n) % r.size
	} else {
		r.Reset()
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
func (r *RingBuffer) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	if r.isEmpty {
		return 0, ErrIsEmpty
	}

	if r.w > r.r {
		n = r.w - r.r
		if n > len(p) {
			n = len(p)
		}
		copy(p, r.buf[r.r:r.r+n])
		r.r += n
		if r.r == r.w {
			r.Reset()
		}
		return
	}

	n = r.size - r.r + r.w
	if n > len(p) {
		n = len(p)
	}

	if r.r+n <= r.size {
		copy(p, r.buf[r.r:r.r+n])
	} else {
		c1 := r.size - r.r
		copy(p, r.buf[r.r:])
		c2 := n - c1
		copy(p[c1:], r.buf[:c2])
	}
	r.r = (r.r + n) % r.size
	if r.r == r.w {
		r.Reset()
	}

	return
}

// ReadByte reads and returns the next byte from the input or ErrIsEmpty.
func (r *RingBuffer) ReadByte() (b byte, err error) {
	if r.isEmpty {
		return 0, ErrIsEmpty
	}
	b = r.buf[r.r]
	r.r++
	if r.r == r.size {
		r.r = 0
	}
	if r.r == r.w {
		r.Reset()
	}

	return
}

// Write writes len(p) bytes from p to the underlying buf.
// It returns the number of bytes written from p (n == len(p) > 0) and any error encountered that caused the write to
// stop early.
// If the length of p is greater than the writable capacity of this ring-buffer, it will allocate more memory to
// this ring-buffer.
// Write must not modify the slice data, even temporarily.
func (r *RingBuffer) Write(p []byte) (n int, err error) {
	n = len(p)
	if n == 0 {
		return
	}

	free := r.Free()
	if n > free {
		r.grow(r.size + n - free)
	}

	if r.w >= r.r {
		c1 := r.size - r.w
		if c1 >= n {
			copy(r.buf[r.w:], p)
			r.w += n
		} else {
			copy(r.buf[r.w:], p[:c1])
			c2 := n - c1
			copy(r.buf, p[c1:])
			r.w = c2
		}
	} else {
		copy(r.buf[r.w:], p)
		r.w += n
	}

	if r.w == r.size {
		r.w = 0
	}

	r.isEmpty = false

	return
}

// WriteByte writes one byte into buffer.
func (r *RingBuffer) WriteByte(c byte) error {
	if r.Free() < 1 {
		r.grow(1)
	}
	r.buf[r.w] = c
	r.w++

	if r.w == r.size {
		r.w = 0
	}
	r.isEmpty = false

	return nil
}

// Length returns the length of available read bytes.
func (r *RingBuffer) Length() int {
	if r.r == r.w {
		if r.isEmpty {
			return 0
		}
		return r.size
	}

	if r.w > r.r {
		return r.w - r.r
	}

	return r.size - r.r + r.w
}

// Len returns the length of the underlying buffer.
func (r *RingBuffer) Len() int {
	return len(r.buf)
}

// Cap returns the size of the underlying buffer.
func (r *RingBuffer) Cap() int {
	return r.size
}

// Free returns the length of available bytes to write.
func (r *RingBuffer) Free() int {
	if r.r == r.w {
		if r.isEmpty {
			return r.size
		}
		return 0
	}

	if r.w < r.r {
		return r.r - r.w
	}

	return r.size - r.w + r.r
}

// WriteString writes the contents of the string s to buffer, which accepts a slice of bytes.
func (r *RingBuffer) WriteString(s string) (int, error) {
	return r.Write(internal.StringToBytes(s))
}

// ByteBuffer returns all available read bytes. It does not move the read pointer and only copy the available data.
func (r *RingBuffer) ByteBuffer() *bytebuffer.ByteBuffer {
	if r.isEmpty {
		return nil
	} else if r.w == r.r {
		bb := bytebuffer.Get()
		_, _ = bb.Write(r.buf[r.r:])
		_, _ = bb.Write(r.buf[:r.w])
		return bb
	}

	bb := bytebuffer.Get()
	if r.w > r.r {
		_, _ = bb.Write(r.buf[r.r:r.w])
		return bb
	}

	_, _ = bb.Write(r.buf[r.r:])

	if r.w != 0 {
		_, _ = bb.Write(r.buf[:r.w])
	}

	return bb
}

// WithByteBuffer combines the available read bytes and the given bytes. It does not move the read pointer and
// only copy the available data.
func (r *RingBuffer) WithByteBuffer(b []byte) *bytebuffer.ByteBuffer {
	if r.isEmpty {
		return &bytebuffer.ByteBuffer{B: b}
	} else if r.w == r.r {
		bb := bytebuffer.Get()
		_, _ = bb.Write(r.buf[r.r:])
		_, _ = bb.Write(r.buf[:r.w])
		_, _ = bb.Write(b)
		return bb
	}

	bb := bytebuffer.Get()
	if r.w > r.r {
		_, _ = bb.Write(r.buf[r.r:r.w])
		_, _ = bb.Write(b)
		return bb
	}

	_, _ = bb.Write(r.buf[r.r:])

	if r.w != 0 {
		_, _ = bb.Write(r.buf[:r.w])
	}
	_, _ = bb.Write(b)

	return bb
}

// IsFull tells if this ring-buffer is full.
func (r *RingBuffer) IsFull() bool {
	return r.r == r.w && !r.isEmpty
}

// IsEmpty tells if this ring-buffer is empty.
func (r *RingBuffer) IsEmpty() bool {
	return r.isEmpty
}

// Reset the read pointer and write pointer to zero.
func (r *RingBuffer) Reset() {
	r.isEmpty = true
	r.r, r.w = 0, 0
}

func (r *RingBuffer) grow(newCap int) {
	if n := r.size; n == 0 {
		if newCap <= defaultBufferSize {
			newCap = defaultBufferSize
		} else {
			newCap = internal.CeilToPowerOfTwo(newCap)
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
	newBuf := make([]byte, newCap)
	oldLen := r.Length()
	_, _ = r.Read(newBuf)
	r.buf = newBuf
	r.r = 0
	r.w = oldLen
	r.size = newCap
}
