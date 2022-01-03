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
	bbPool "github.com/panjf2000/gnet/v2/pkg/pool/bytebuffer"
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
	return b.instance().Peek(n)
}

// Discard skips the next n bytes by advancing the read pointer.
func (b *RingBuffer) Discard(n int) (discarded int, err error) {
	defer b.done()
	return b.instance().Discard(n)
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
func (b *RingBuffer) Read(p []byte) (n int, err error) {
	defer b.done()
	return b.instance().Read(p)
}

// ReadByte reads and returns the next byte from the input or ErrIsEmpty.
func (b *RingBuffer) ReadByte() (byte, error) {
	defer b.done()
	return b.instance().ReadByte()
}

// Write writes len(p) bytes from p to the underlying buf.
// It returns the number of bytes written from p (n == len(p) > 0) and any error encountered that caused the write to
// stop early.
// If the length of p is greater than the writable capacity of this ring-buffer, it will allocate more memory to
// this ring-buffer.
// Write must not modify the slice data, even temporarily.
func (b *RingBuffer) Write(p []byte) (n int, err error) {
	return b.instance().Write(p)
}

// WriteByte writes one byte into buffer.
func (b *RingBuffer) WriteByte(c byte) error {
	return b.instance().WriteByte(c)
}

// Buffered returns the length of available bytes to read.
func (b *RingBuffer) Buffered() int {
	return b.instance().Buffered()
}

// Len returns the length of the underlying buffer.
func (b *RingBuffer) Len() int {
	return b.instance().Len()
}

// Cap returns the size of the underlying buffer.
func (b *RingBuffer) Cap() int {
	return b.instance().Cap()
}

// Available returns the length of available bytes to write.
func (b *RingBuffer) Available() int {
	return b.instance().Available()
}

// WriteString writes the contents of the string s to buffer, which accepts a slice of bytes.
func (b *RingBuffer) WriteString(s string) (int, error) {
	return b.instance().WriteString(s)
}

// ByteBuffer returns all available read bytes. It does not move the read pointer and only copy the available data.
func (b *RingBuffer) ByteBuffer() *bbPool.ByteBuffer {
	return b.instance().ByteBuffer()
}

// WithByteBuffer combines the available read bytes and the given bytes. It does not move the read pointer and
// only copy the available data.
func (b *RingBuffer) WithByteBuffer(p []byte) *bbPool.ByteBuffer {
	defer b.done()
	return b.instance().WithByteBuffer(p)
}

// ReadFrom implements io.ReaderFrom.
func (b *RingBuffer) ReadFrom(r io.Reader) (n int64, err error) {
	return b.instance().ReadFrom(r)
}

// WriteTo implements io.WriterTo.
func (b *RingBuffer) WriteTo(w io.Writer) (int64, error) {
	defer b.done()
	return b.instance().WriteTo(w)
}

// IsFull tells if this ring-buffer is full.
func (b *RingBuffer) IsFull() bool {
	return b.instance().IsFull()
}

// IsEmpty tells if this ring-buffer is empty.
func (b *RingBuffer) IsEmpty() bool {
	return b.instance().IsEmpty()
}

// Reset the read pointer and write pointer to zero.
func (b *RingBuffer) Reset() {
	b.instance().Reset()
	b.done()
}
