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

package ring

import (
	"errors"
	"io"

	"github.com/panjf2000/gnet/v2/internal/toolkit"
	bsPool "github.com/panjf2000/gnet/v2/pkg/pool/byteslice"
)

const (
	// MinRead is the minimum slice size passed to a Read call by
	// Buffer.ReadFrom. As long as the Buffer has at least MinRead bytes beyond
	// what is required to hold the contents of r, ReadFrom will not grow the
	// underlying buffer.
	MinRead = 512
	// DefaultBufferSize is the first-time allocation on a ring-buffer.
	DefaultBufferSize   = 1024     // 1KB
	bufferGrowThreshold = 4 * 1024 // 4KB
)

// ErrIsEmpty will be returned when trying to read an empty ring-buffer.
var ErrIsEmpty = errors.New("ring-buffer is empty")

// Buffer is a circular buffer that implement io.ReaderWriter interface.
type Buffer struct {
	bs      [][]byte
	buf     []byte
	size    int
	r       int // next position to read
	w       int // next position to write
	isEmpty bool
}

// New returns a new Buffer whose buffer has the given size.
func New(size int) *Buffer {
	if size == 0 {
		return &Buffer{bs: make([][]byte, 2), isEmpty: true}
	}
	size = toolkit.CeilToPowerOfTwo(size)
	return &Buffer{
		bs:      make([][]byte, 2),
		buf:     make([]byte, size),
		size:    size,
		isEmpty: true,
	}
}

// Peek returns the next n bytes without advancing the read pointer,
// it returns all bytes when n <= 0.
func (rb *Buffer) Peek(n int) (head []byte, tail []byte) {
	if rb.isEmpty {
		return
	}

	if n <= 0 {
		return rb.peekAll()
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

// peekAll returns all bytes without advancing the read pointer.
func (rb *Buffer) peekAll() (head []byte, tail []byte) {
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
func (rb *Buffer) Discard(n int) (discarded int, err error) {
	if n <= 0 {
		return 0, nil
	}

	discarded = rb.Buffered()
	if n < discarded {
		rb.r = (rb.r + n) % rb.size
		return n, nil
	}
	rb.Reset()
	return
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
func (rb *Buffer) Read(p []byte) (n int, err error) {
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
func (rb *Buffer) ReadByte() (b byte, err error) {
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
func (rb *Buffer) Write(p []byte) (n int, err error) {
	n = len(p)
	if n == 0 {
		return
	}

	free := rb.Available()
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
func (rb *Buffer) WriteByte(c byte) error {
	if rb.Available() < 1 {
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

// Buffered returns the length of available bytes to read.
func (rb *Buffer) Buffered() int {
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
func (rb *Buffer) Len() int {
	return len(rb.buf)
}

// Cap returns the size of the underlying buffer.
func (rb *Buffer) Cap() int {
	return rb.size
}

// Available returns the length of available bytes to write.
func (rb *Buffer) Available() int {
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
func (rb *Buffer) WriteString(s string) (int, error) {
	return rb.Write(toolkit.StringToBytes(s))
}

// Bytes returns all available read bytes. It does not move the read pointer and only copy the available data.
func (rb *Buffer) Bytes() []byte {
	if rb.isEmpty {
		return nil
	} else if rb.w == rb.r {
		var bb []byte
		bb = append(bb, rb.buf[rb.r:]...)
		bb = append(bb, rb.buf[:rb.w]...)
		return bb
	}

	var bb []byte
	if rb.w > rb.r {
		bb = append(bb, rb.buf[rb.r:rb.w]...)
		return bb
	}

	bb = append(bb, rb.buf[rb.r:]...)

	if rb.w != 0 {
		bb = append(bb, rb.buf[:rb.w]...)
	}

	return bb
}

// ReadFrom implements io.ReaderFrom.
func (rb *Buffer) ReadFrom(r io.Reader) (n int64, err error) {
	var m int
	for {
		if rb.Available() < MinRead {
			rb.grow(rb.Buffered() + MinRead)
		}

		if rb.w >= rb.r {
			m, err = r.Read(rb.buf[rb.w:])
			if m < 0 {
				panic("RingBuffer.ReadFrom: reader returned negative count from Read")
			}
			rb.isEmpty = false
			rb.w = (rb.w + m) % rb.size
			n += int64(m)
			if err == io.EOF {
				return n, nil
			}
			if err != nil {
				return
			}
			m, err = r.Read(rb.buf[:rb.r])
			if m < 0 {
				panic("RingBuffer.ReadFrom: reader returned negative count from Read")
			}
			rb.w = (rb.w + m) % rb.size
			n += int64(m)
			if err == io.EOF {
				return n, nil
			}
			if err != nil {
				return
			}
		} else {
			m, err = r.Read(rb.buf[rb.w:rb.r])
			if m < 0 {
				panic("RingBuffer.ReadFrom: reader returned negative count from Read")
			}
			rb.isEmpty = false
			rb.w = (rb.w + m) % rb.size
			n += int64(m)
			if err == io.EOF {
				return n, nil
			}
			if err != nil {
				return
			}
		}
	}
}

// WriteTo implements io.WriterTo.
func (rb *Buffer) WriteTo(w io.Writer) (int64, error) {
	if rb.isEmpty {
		return 0, ErrIsEmpty
	}

	if rb.w > rb.r {
		n := rb.w - rb.r
		m, err := w.Write(rb.buf[rb.r : rb.r+n])
		if m > n {
			panic("RingBuffer.WriteTo: invalid Write count")
		}
		rb.r += m
		if rb.r == rb.w {
			rb.Reset()
		}
		if err != nil {
			return int64(m), err
		}
		if !rb.isEmpty {
			return int64(m), io.ErrShortWrite
		}
		return int64(m), nil
	}

	n := rb.size - rb.r + rb.w
	if rb.r+n <= rb.size {
		m, err := w.Write(rb.buf[rb.r : rb.r+n])
		if m > n {
			panic("RingBuffer.WriteTo: invalid Write count")
		}
		rb.r = (rb.r + m) % rb.size
		if rb.r == rb.w {
			rb.Reset()
		}
		if err != nil {
			return int64(m), err
		}
		if !rb.isEmpty {
			return int64(m), io.ErrShortWrite
		}
		return int64(m), nil
	}

	var cum int64
	c1 := rb.size - rb.r
	m, err := w.Write(rb.buf[rb.r:])
	if m > c1 {
		panic("RingBuffer.WriteTo: invalid Write count")
	}
	rb.r = (rb.r + m) % rb.size
	if err != nil {
		return int64(m), err
	}
	if m < c1 {
		return int64(m), io.ErrShortWrite
	}
	cum += int64(m)
	c2 := n - c1
	m, err = w.Write(rb.buf[:c2])
	if m > c2 {
		panic("RingBuffer.WriteTo: invalid Write count")
	}
	rb.r = m
	cum += int64(m)
	if rb.r == rb.w {
		rb.Reset()
	}
	if err != nil {
		return cum, err
	}
	if !rb.isEmpty {
		return cum, io.ErrShortWrite
	}
	return cum, nil
}

// IsFull tells if this ring-buffer is full.
func (rb *Buffer) IsFull() bool {
	return rb.r == rb.w && !rb.isEmpty
}

// IsEmpty tells if this ring-buffer is empty.
func (rb *Buffer) IsEmpty() bool {
	return rb.isEmpty
}

// Reset the read pointer and write pointer to zero.
func (rb *Buffer) Reset() {
	rb.isEmpty = true
	rb.r, rb.w = 0, 0
}

func (rb *Buffer) grow(newCap int) {
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
	newBuf := bsPool.Get(newCap)
	oldLen := rb.Buffered()
	_, _ = rb.Read(newBuf)
	bsPool.Put(rb.buf)
	rb.buf = newBuf
	rb.r = 0
	rb.w = oldLen
	rb.size = newCap
	if rb.w > 0 {
		rb.isEmpty = false
	}
}
