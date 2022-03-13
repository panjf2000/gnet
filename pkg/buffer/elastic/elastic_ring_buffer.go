// Copyright (c) 2022 Andy Pan
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

package elastic

import (
	"io"

	"github.com/panjf2000/gnet/v2/pkg/buffer/ring"
	rbPool "github.com/panjf2000/gnet/v2/pkg/pool/ringbuffer"
)

// RingBuffer is the elastic wrapper of ring.Buffer.
type RingBuffer struct {
	rb *ring.Buffer
}

func (b *RingBuffer) instance() *ring.Buffer {
	if b.rb == nil {
		b.rb = rbPool.Get()
	}

	return b.rb
}

// Done checks and returns the internal ring-buffer to pool.
func (b *RingBuffer) Done() {
	if b.rb != nil {
		rbPool.Put(b.rb)
		b.rb = nil
	}
}

func (b *RingBuffer) done() {
	if b.rb != nil && b.rb.IsEmpty() {
		rbPool.Put(b.rb)
		b.rb = nil
	}
}

// Peek returns the next n bytes without advancing the read pointer,
// it returns all bytes when n <= 0.
func (b *RingBuffer) Peek(n int) (head []byte, tail []byte) {
	if b.rb == nil {
		return nil, nil
	}
	return b.rb.Peek(n)
}

// Discard skips the next n bytes by advancing the read pointer.
func (b *RingBuffer) Discard(n int) (int, error) {
	if b.rb == nil {
		return 0, ring.ErrIsEmpty
	}

	defer b.done()
	return b.rb.Discard(n)
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
func (b *RingBuffer) Read(p []byte) (int, error) {
	if b.rb == nil {
		return 0, ring.ErrIsEmpty
	}

	defer b.done()
	return b.rb.Read(p)
}

// ReadByte reads and returns the next byte from the input or ErrIsEmpty.
func (b *RingBuffer) ReadByte() (byte, error) {
	if b.rb == nil {
		return 0, ring.ErrIsEmpty
	}

	defer b.done()
	return b.rb.ReadByte()
}

// Write writes len(p) bytes from p to the underlying buf.
// It returns the number of bytes written from p (n == len(p) > 0) and any error encountered that caused the write to
// stop early.
// If the length of p is greater than the writable capacity of this ring-buffer, it will allocate more memory to
// this ring-buffer.
// Write must not modify the slice data, even temporarily.
func (b *RingBuffer) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return b.instance().Write(p)
}

// WriteByte writes one byte into buffer.
func (b *RingBuffer) WriteByte(c byte) error {
	return b.instance().WriteByte(c)
}

// Buffered returns the length of available bytes to read.
func (b *RingBuffer) Buffered() int {
	if b.rb == nil {
		return 0
	}
	return b.rb.Buffered()
}

// Len returns the length of the underlying buffer.
func (b *RingBuffer) Len() int {
	if b.rb == nil {
		return 0
	}
	return b.rb.Len()
}

// Cap returns the size of the underlying buffer.
func (b *RingBuffer) Cap() int {
	if b.rb == nil {
		return 0
	}
	return b.rb.Cap()
}

// Available returns the length of available bytes to write.
func (b *RingBuffer) Available() int {
	if b.rb == nil {
		return 0
	}
	return b.rb.Available()
}

// WriteString writes the contents of the string s to buffer, which accepts a slice of bytes.
func (b *RingBuffer) WriteString(s string) (int, error) {
	if len(s) == 0 {
		return 0, nil
	}
	return b.instance().WriteString(s)
}

// Bytes returns all available read bytes. It does not move the read pointer and only copy the available data.
func (b *RingBuffer) Bytes() []byte {
	if b.rb == nil {
		return nil
	}
	return b.rb.Bytes()
}

// ReadFrom implements io.ReaderFrom.
func (b *RingBuffer) ReadFrom(r io.Reader) (int64, error) {
	return b.instance().ReadFrom(r)
}

// WriteTo implements io.WriterTo.
func (b *RingBuffer) WriteTo(w io.Writer) (int64, error) {
	if b.rb == nil {
		return 0, ring.ErrIsEmpty
	}

	defer b.done()
	return b.instance().WriteTo(w)
}

// IsFull tells if this ring-buffer is full.
func (b *RingBuffer) IsFull() bool {
	if b.rb == nil {
		return false
	}
	return b.rb.IsFull()
}

// IsEmpty tells if this ring-buffer is empty.
func (b *RingBuffer) IsEmpty() bool {
	if b.rb == nil {
		return true
	}
	return b.rb.IsEmpty()
}

// Reset the read pointer and write pointer to zero.
func (b *RingBuffer) Reset() {
	if b.rb == nil {
		return
	}
	b.rb.Reset()
}
