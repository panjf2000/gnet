// Copyright (c) 2019 Chao yuepan, Andy Pan
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
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRingBuffer_Write(t *testing.T) {
	rb := New(64)

	_, err := rb.ReadByte()
	assert.ErrorIs(t, err, ErrIsEmpty, "expect nil err, but got nil")
	// check empty or full
	assert.True(t, rb.IsEmpty(), "expect IsEmpty is true but got false")
	assert.False(t, rb.IsFull(), "expect IsFull is false but got true")
	assert.EqualValuesf(t, 0, rb.Length(), "expect len 0 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	assert.EqualValuesf(t, 64, rb.Free(), "expect free 64 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)

	// write 4 * 4 = 16 bytes
	data := []byte(strings.Repeat("abcd", 4))
	n, _ := rb.Write(data)
	assert.EqualValuesf(t, 16, n, "expect write 16 bytes but got %d", n)
	assert.EqualValuesf(t, 16, rb.Length(), "expect len 16 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	assert.EqualValuesf(t, 48, rb.Free(), "expect free 48 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	assert.EqualValuesf(t, data, rb.ByteBuffer().Bytes(), "expect 4 abcd but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)

	// check empty or full
	assert.False(t, rb.IsEmpty(), "expect IsEmpty is false but got true")
	assert.False(t, rb.IsFull(), "expect IsFull is false but got true")

	// write 48 bytes, should full
	data = []byte(strings.Repeat("abcd", 12))
	n, _ = rb.Write(data)
	assert.EqualValuesf(t, 48, n, "expect write 48 bytes but got %d", n)
	assert.EqualValuesf(t, 64, rb.Length(), "expect len 64 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	assert.EqualValuesf(t, 0, rb.Free(), "expect free 0 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	assert.EqualValuesf(t, 0, rb.w, "expect r.w=0 but got %d. r.r=%d", rb.w, rb.r)
	assert.EqualValuesf(t, []byte(strings.Repeat("abcd", 16)), rb.ByteBuffer().Bytes(), "expect 16 abcd but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)

	// check empty or full
	assert.False(t, rb.IsEmpty(), "expect IsEmpty is false but got true")
	assert.True(t, rb.IsFull(), "expect IsFull is true but got false")
	assert.True(t, rb.IsFull(), "expect IsFull is true but got false")

	// write more 4 bytes, should scale from 64 to 128 bytes.
	n, _ = rb.Write([]byte(strings.Repeat("abcd", 1)))
	assert.EqualValuesf(t, 4, n, "expect write 4 bytes but got %d", n)
	size := rb.Cap()
	assert.EqualValuesf(t, 68, rb.Length(), "expect len 68 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	assert.EqualValuesf(t, size-68, rb.Free(), "expect free 60 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)

	// check empty or full
	assert.False(t, rb.IsEmpty(), "expect IsEmpty is false but got true")
	assert.False(t, rb.IsFull(), "expect IsFull is false but got true")

	// reset this ringbuffer and set a long slice
	rb.Reset()
	n, _ = rb.Write([]byte(strings.Repeat("abcd", 20)))
	assert.EqualValuesf(t, 80, n, "expect write 80 bytes but got %d", n)
	assert.EqualValuesf(t, 80, rb.Length(), "expect len 80 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	assert.EqualValuesf(t, size-80, rb.Free(), "expect free 48 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	assert.Greaterf(t, rb.w, 0, "expect r.w>=0 but got %d. r.r=%d", rb.w, rb.r)

	// check empty or full
	assert.False(t, rb.IsEmpty(), "expect IsEmpty is false but got true")
	assert.False(t, rb.IsFull(), "expect IsFull is false but got true")

	assert.EqualValuesf(t, []byte(strings.Repeat("abcd", 20)), rb.ByteBuffer().Bytes(), "expect 16 abcd but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)

	rb.Reset()
	size = rb.Cap()
	// write 4 * 2 = 8 bytes
	n, _ = rb.Write([]byte(strings.Repeat("abcd", 2)))
	assert.EqualValuesf(t, 8, n, "expect write 16 bytes but got %d", n)
	assert.EqualValuesf(t, 8, rb.Length(), "expect len 16 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	assert.EqualValuesf(t, size-8, rb.Free(), "expect free 48 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	buf := make([]byte, 5)
	_, _ = rb.Read(buf)
	assert.EqualValuesf(t, 3, rb.Length(), "expect len 3 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	_, _ = rb.Write([]byte(strings.Repeat("abcd", 15)))

	assert.EqualValuesf(t, []byte("bcd"+strings.Repeat("abcd", 15)), rb.ByteBuffer().Bytes(), "expect 63 ... but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)

	rb.Reset()
	n, _ = rb.Write([]byte(strings.Repeat("abcd", 32)))
	assert.EqualValuesf(t, 128, n, "expect write 128 bytes but got %d", n)
	assert.EqualValuesf(t, 0, rb.Free(), "expect free 0 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	buf = make([]byte, 16)
	_, _ = rb.Read(buf)
	n, _ = rb.Write([]byte(strings.Repeat("1234", 4)))
	assert.EqualValuesf(t, 16, n, "expect write 16 bytes but got %d", n)
	assert.EqualValuesf(t, 0, rb.Free(), "expect free 0 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	assert.EqualValuesf(t, []byte(strings.Repeat("abcd", 32)+strings.Repeat("1234", 4)), append(buf, rb.ByteBuffer().Bytes()...), "expect 16 abcd and 4 1234 but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)
}

func TestZeroRingBuffer(t *testing.T) {
	rb := New(0)
	head, tail := rb.Peek(2)
	assert.Empty(t, head, "head should be empty")
	assert.Empty(t, tail, "tail should be empty")
	head, tail = rb.PeekAll()
	assert.Empty(t, head, "head should be empty")
	assert.Empty(t, tail, "tail should be empty")
	assert.EqualValues(t, 0, rb.Length(), "expect length is 0")
	assert.EqualValues(t, 0, rb.Free(), "expect free is 0")
	buf := []byte(strings.Repeat("1234", 12))
	_, _ = rb.Write(buf)
	assert.EqualValuesf(t, defaultBufferSize, rb.Len(), "expect rb.Len()=%d, but got rb.Len()=%d", defaultBufferSize, rb.Len())
	assert.EqualValuesf(t, defaultBufferSize, rb.Cap(), "expect rb.Cap()=%d, but got rb.Cap()=%d", defaultBufferSize, rb.Cap())
	assert.Truef(t, rb.r == 0 && rb.w == 48 && rb.size == defaultBufferSize, "expect rb.r=0, rb.w=48, rb.size=64, rb.mask=63, but got rb.r=%d, rb.w=%d, rb.size=%d", rb.r, rb.w, rb.size)
	assert.EqualValues(t, buf, rb.ByteBuffer().Bytes(), "expect it is equal")
	rb.Discard(48)
	assert.Truef(t, rb.IsEmpty() && rb.r == 0 && rb.w == 0, "expect rb is empty and rb.r=rb.w=0, but got rb.r=%d and rb.w=%d", rb.r, rb.w)
}

func TestRingBufferGrow(t *testing.T) {
	rb := New(0)
	head, tail := rb.Peek(2)
	assert.Empty(t, head, "head should be empty")
	assert.Empty(t, tail, "tail should be empty")
	data := make([]byte, defaultBufferSize+1)
	n, err := rand.Read(data)
	assert.NoError(t, err, "failed to generate random data")
	assert.EqualValuesf(t, defaultBufferSize+1, n, "expect random data length is %d but got %d", defaultBufferSize+1, n)
	n, err = rb.Write(data)
	assert.NoError(t, err)
	assert.EqualValues(t, defaultBufferSize+1, n)
	assert.EqualValues(t, 2*defaultBufferSize, rb.Cap())
	assert.EqualValues(t, 2*defaultBufferSize, rb.Len())
	assert.EqualValues(t, defaultBufferSize+1, rb.Length())
	assert.EqualValues(t, defaultBufferSize-1, rb.Free())
	assert.EqualValues(t, data, rb.ByteBuffer().Bytes())

	rb = New(defaultBufferSize)
	newData := make([]byte, 3*512)
	n, err = rand.Read(newData)
	assert.NoError(t, err, "failed to generate random data")
	assert.EqualValuesf(t, 3*512, n, "expect random data length is %d but got %d", 3*512, n)
	n, err = rb.Write(newData)
	assert.NoError(t, err)
	assert.EqualValues(t, 3*512, n)
	assert.EqualValues(t, 2*defaultBufferSize, rb.Cap())
	assert.EqualValues(t, 2*defaultBufferSize, rb.Len())
	assert.EqualValues(t, 3*512, rb.Length())
	assert.EqualValues(t, 512, rb.Free())
	assert.EqualValues(t, newData, rb.ByteBuffer().Bytes())

	rb.Reset()
	data = make([]byte, bufferGrowThreshold)
	n, err = rand.Read(data)
	assert.NoError(t, err, "failed to generate random data")
	assert.EqualValuesf(t, bufferGrowThreshold, n, "expect random data length is %d but got %d", bufferGrowThreshold, n)
	n, err = rb.Write(data)
	assert.NoError(t, err)
	assert.EqualValues(t, bufferGrowThreshold, n)
	assert.EqualValues(t, bufferGrowThreshold, rb.Cap())
	assert.EqualValues(t, bufferGrowThreshold, rb.Len())
	assert.EqualValues(t, bufferGrowThreshold, rb.Length())
	assert.EqualValues(t, 0, rb.Free())
	assert.EqualValues(t, data, rb.ByteBuffer().Bytes())
	newData = make([]byte, bufferGrowThreshold/2+1)
	n, err = rand.Read(newData)
	assert.NoError(t, err, "failed to generate random data")
	assert.EqualValuesf(t, bufferGrowThreshold/2+1, n, "expect random data length is %d but got %d", bufferGrowThreshold, n)
	n, err = rb.Write(newData)
	assert.NoError(t, err)
	assert.EqualValues(t, bufferGrowThreshold/2+1, n)
	assert.EqualValues(t, 1.25*(1.25*bufferGrowThreshold), rb.Cap())
	assert.EqualValues(t, 1.25*(1.25*bufferGrowThreshold), rb.Len())
	assert.EqualValues(t, 1.5*bufferGrowThreshold+1, rb.Length())
	assert.EqualValues(t, 1.25*(1.25*bufferGrowThreshold)-rb.Length(), rb.Free())
	assert.EqualValues(t, append(data, newData...), rb.ByteBuffer().Bytes())
}

func TestRingBuffer_Read(t *testing.T) {
	rb := New(64)

	// check empty or full
	assert.True(t, rb.IsEmpty(), "expect IsEmpty is true but got false")
	assert.False(t, rb.IsFull(), "expect isfull is false but got true")
	assert.EqualValuesf(t, 0, rb.Length(), "expect len 0 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	assert.EqualValuesf(t, 64, rb.Free(), "expect free 64 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)

	// read empty
	buf := make([]byte, 1024)
	n, err := rb.Read(buf)
	assert.ErrorIs(t, err, ErrIsEmpty, "expect ErrIsEmpty but got nil")
	assert.EqualValuesf(t, 0, n, "expect read 0 bytes but got %d", n)
	assert.EqualValuesf(t, 0, rb.Length(), "expect len 0 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	assert.EqualValuesf(t, 64, rb.Free(), "expect free 64 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	assert.EqualValuesf(t, 0, rb.r, "expect r.r=0 but got %d. r.w=%d", rb.r, rb.w)

	// write 16 bytes to read
	_, _ = rb.Write([]byte(strings.Repeat("abcd", 4)))
	// read all data from buffer, it will be shrunk from 64 to 32.
	n, err = rb.Read(buf)
	assert.NoErrorf(t, err, "read failed: %v", err)
	assert.EqualValuesf(t, 16, n, "expect read 16 bytes but got %d", n)
	assert.EqualValuesf(t, 0, rb.Length(), "expect len 0 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	assert.EqualValuesf(t, 64, rb.Free(), "expect free 64 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	assert.EqualValuesf(t, 0, rb.r, "expect r.r=0 but got %d. r.w=%d", rb.r, rb.w)

	// write long slice to read, it will scale from 32 to 128 bytes.
	_, _ = rb.Write([]byte(strings.Repeat("abcd", 20)))
	// read all data from buffer, it will be shrunk from 128 to 64.
	n, err = rb.Read(buf)
	assert.NoErrorf(t, err, "read failed: %v", err)
	assert.EqualValuesf(t, 80, n, "expect read 80 bytes but got %d", n)
	assert.EqualValuesf(t, 0, rb.Length(), "expect len 0 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	assert.EqualValuesf(t, 128, rb.Free(), "expect free 128 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	assert.EqualValuesf(t, 0, rb.r, "expect r.r=0 but got %d. r.w=%d", rb.r, rb.w)

	rb.Reset()
	_, _ = rb.Write([]byte(strings.Repeat("1234", 32)))
	assert.True(t, rb.IsFull(), "ring buffer should be full")
	assert.EqualValuesf(t, 0, rb.Free(), "expect free 0 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	assert.EqualValuesf(t, 0, rb.w, "expect r.2=0 but got %d. r.r=%d", rb.w, rb.r)
	head, tail := rb.Peek(64)
	assert.Truef(t, len(head) == 64 && tail == nil, "expect len(head)=64 and tail=nil, yet len(head)=%d and tail != nil", len(head))
	assert.EqualValuesf(t, 0, rb.r, "expect r.r=0 but got %d", rb.r)
	assert.EqualValues(t, []byte(strings.Repeat("1234", 16)), head)
	rb.Discard(64)
	assert.EqualValuesf(t, 64, rb.r, "expect r.r=64 but got %d", rb.r)
	_, _ = rb.Write([]byte(strings.Repeat("1234", 4)))
	assert.EqualValuesf(t, 16, rb.w, "expect r.w=16 but got %d", rb.w)
	head, tail = rb.Peek(128)
	assert.Truef(t, len(head) == 64 && len(tail) == 16, "expect len(head)=64 and len(tail)=16, yet len(head)=%d and len(tail)=%d", len(head), len(tail))
	assert.EqualValues(t, []byte(strings.Repeat("1234", 16)), head)
	assert.EqualValues(t, []byte(strings.Repeat("1234", 4)), tail)

	head, tail = rb.PeekAll()
	assert.Truef(t, len(head) == 64 && len(tail) == 16, "expect len(head)=64 and len(tail)=16, yet len(head)=%d and len(tail)=%d", len(head), len(tail))
	assert.EqualValues(t, []byte(strings.Repeat("1234", 16)), head)
	assert.EqualValues(t, []byte(strings.Repeat("1234", 4)), tail)

	rb.Discard(64)
	rb.Discard(16)
	assert.True(t, rb.IsEmpty(), "should be empty")
}

func TestRingBuffer_ByteInterface(t *testing.T) {
	rb := New(2)

	// write one
	_ = rb.WriteByte('a')
	assert.EqualValuesf(t, 1, rb.Length(), "expect len 1 byte but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	assert.EqualValuesf(t, 1, rb.Free(), "expect free 1 byte but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	assert.EqualValuesf(t, []byte{'a'}, rb.ByteBuffer().Bytes(), "expect a but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)
	// check empty or full
	assert.Falsef(t, rb.IsEmpty(), "expect IsEmpty is false but got true")
	assert.False(t, rb.IsFull(), "expect IsFull is false but got true")

	// write two, isFull
	_ = rb.WriteByte('b')
	assert.EqualValuesf(t, 2, rb.Length(), "expect len 2 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	assert.EqualValuesf(t, 0, rb.Free(), "expect free 0 byte but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	assert.EqualValuesf(t, []byte{'a', 'b'}, rb.ByteBuffer().Bytes(), "expect a but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)
	// check empty or full
	assert.False(t, rb.IsEmpty(), "expect IsEmpty is false but got true")
	assert.True(t, rb.IsFull(), "expect IsFull is true but got false")

	// write, it will scale from 2 to 4 bytes.
	_ = rb.WriteByte('c')
	assert.EqualValuesf(t, 3, rb.Length(), "expect len 3 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	assert.EqualValuesf(t, 1, rb.Free(), "expect free 1 byte but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	assert.EqualValuesf(t, []byte{'a', 'b', 'c'}, rb.ByteBuffer().Bytes(), "expect a but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)
	// check empty or full
	assert.False(t, rb.IsEmpty(), "expect IsEmpty is false but got true")
	assert.False(t, rb.IsFull(), "expect IsFull is false but got true")

	// read one
	b, err := rb.ReadByte()
	assert.NoErrorf(t, err, "ReadByte failed: %v", err)
	assert.EqualValuesf(t, 'a', b, "expect a but got %c. r.w=%d, r.r=%d", b, rb.w, rb.r)
	assert.EqualValuesf(t, 2, rb.Length(), "expect len 2 byte but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	assert.EqualValuesf(t, 2, rb.Free(), "expect free 2 byte but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	assert.EqualValuesf(t, []byte{'b', 'c'}, rb.ByteBuffer().Bytes(), "expect a but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)
	// check empty or full
	assert.False(t, rb.IsEmpty(), "expect IsEmpty is false but got true")
	assert.False(t, rb.IsFull(), "expect IsFull is false but got true")

	// read two
	b, _ = rb.ReadByte()
	assert.EqualValuesf(t, 'b', b, "expect b but got %c. r.w=%d, r.r=%d", b, rb.w, rb.r)
	assert.EqualValuesf(t, 1, rb.Length(), "expect len 1 byte but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	assert.EqualValuesf(t, 3, rb.Free(), "expect free 3 byte but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	// check empty or full
	assert.False(t, rb.IsEmpty(), "expect IsEmpty is false but got true")
	assert.False(t, rb.IsFull(), "expect IsFull is false but got true")

	// read three
	_, _ = rb.ReadByte()
	assert.EqualValuesf(t, 0, rb.Length(), "expect len 0 byte but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	assert.EqualValuesf(t, 4, rb.Free(), "expect free 4 byte but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	// check empty or full
	assert.True(t, rb.IsEmpty(), "expect IsEmpty is true but got false")
	assert.False(t, rb.IsFull(), "expect IsFull is false but got true")
}
