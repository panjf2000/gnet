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
	"bytes"
	"strings"
	"testing"
)

func TestRingBuffer_Write(t *testing.T) {
	rb := New(64)

	if _, err := rb.ReadByte(); err == nil {
		t.Fatal("expect empty err, but got nil")
	}
	// check empty or full
	if !rb.IsEmpty() {
		t.Fatalf("expect IsEmpty is true but got false")
	}
	if rb.IsFull() {
		t.Fatalf("expect IsFull is false but got true")
	}
	if rb.Length() != 0 {
		t.Fatalf("expect len 0 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	}
	if rb.Free() != 64 {
		t.Fatalf("expect free 64 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}

	// write 4 * 4 = 16 bytes
	n, _ := rb.Write([]byte(strings.Repeat("abcd", 4)))
	if n != 16 {
		t.Fatalf("expect write 16 bytes but got %d", n)
	}
	if rb.Length() != 16 {
		t.Fatalf("expect len 16 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	}
	if rb.Free() != 48 {
		t.Fatalf("expect free 48 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}
	if !bytes.Equal(rb.ByteBuffer().Bytes(), []byte(strings.Repeat("abcd", 4))) {
		t.Fatalf("expect 4 abcd but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)
	}

	// check empty or full
	if rb.IsEmpty() {
		t.Fatalf("expect IsEmpty is false but got true")
	}
	if rb.IsFull() {
		t.Fatalf("expect IsFull is false but got true")
	}

	// write 48 bytes, should full
	n, _ = rb.Write([]byte(strings.Repeat("abcd", 12)))
	if n != 48 {
		t.Fatalf("expect write 48 bytes but got %d", n)
	}
	if rb.Length() != 64 {
		t.Fatalf("expect len 64 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	}
	if rb.Free() != 0 {
		t.Fatalf("expect free 0 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}
	if rb.w != 0 {
		t.Fatalf("expect r.w=0 but got %d. r.r=%d", rb.w, rb.r)
	}
	if !bytes.Equal(rb.ByteBuffer().Bytes(), []byte(strings.Repeat("abcd", 16))) {
		t.Fatalf("expect 16 abcd but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)
	}

	// check empty or full
	if rb.IsEmpty() {
		t.Fatalf("expect IsEmpty is false but got true")
	}
	if !rb.IsFull() {
		t.Fatalf("expect IsFull is true but got false")
	}

	// write more 4 bytes, should scale from 64 to 128 bytes.
	n, _ = rb.Write([]byte(strings.Repeat("abcd", 1)))
	size := rb.Cap()
	if n != 4 {
		t.Fatalf("expect write 4 bytes but got %d", n)
	}
	if rb.Length() != 68 {
		t.Fatalf("expect len 68 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	}
	if rb.Free() != size-68 {
		t.Fatalf("expect free 60 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}

	// check empty or full
	if rb.IsEmpty() {
		t.Fatalf("expect IsEmpty is false but got true")
	}
	if rb.IsFull() {
		t.Fatalf("expect IsFull is false but got true")
	}

	// reset this ringbuffer and set a long slice
	rb.Reset()
	n, _ = rb.Write([]byte(strings.Repeat("abcd", 20)))
	if n != 80 {
		t.Fatalf("expect write 80 bytes but got %d", n)
	}
	if rb.Length() != 80 {
		t.Fatalf("expect len 80 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	}
	if rb.Free() != size-80 {
		t.Fatalf("expect free 48 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}
	if rb.w == 0 {
		t.Fatalf("expect r.w!=0 but got %d. r.r=%d", rb.w, rb.r)
	}

	// check empty or full
	if rb.IsEmpty() {
		t.Fatalf("expect IsEmpty is false but got true")
	}
	if rb.IsFull() {
		t.Fatalf("expect IsFull is false but got true")
	}

	if !bytes.Equal(rb.ByteBuffer().Bytes(), []byte(strings.Repeat("abcd", 20))) {
		t.Fatalf("expect 16 abcd but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)
	}

	rb.Reset()
	size = rb.Cap()
	// write 4 * 2 = 8 bytes
	n, _ = rb.Write([]byte(strings.Repeat("abcd", 2)))
	if n != 8 {
		t.Fatalf("expect write 16 bytes but got %d", n)
	}
	if rb.Length() != 8 {
		t.Fatalf("expect len 16 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	}
	if rb.Free() != size-8 {
		t.Fatalf("expect free 48 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}
	buf := make([]byte, 5)
	_, _ = rb.Read(buf)
	if rb.Length() != 3 {
		t.Fatalf("expect len 3 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	}
	_, _ = rb.Write([]byte(strings.Repeat("abcd", 15)))

	if !bytes.Equal(rb.ByteBuffer().Bytes(), []byte("bcd"+strings.Repeat("abcd", 15))) {
		t.Fatalf("expect 63 ... but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)
	}

	rb.Reset()
	n, _ = rb.Write([]byte(strings.Repeat("abcd", 32)))
	if n != 128 {
		t.Fatalf("expect write 128 bytes but got %d", n)
	}
	if rb.Free() != 0 {
		t.Fatalf("expect free 0 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}
	buf = make([]byte, 16)
	_, _ = rb.Read(buf)
	n, _ = rb.Write([]byte(strings.Repeat("1234", 4)))
	if n != 16 {
		t.Fatalf("expect write 16 bytes but got %d", n)
	}
	if rb.Free() != 0 {
		t.Fatalf("expect free 0 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}
	if !bytes.Equal(append(buf, rb.ByteBuffer().Bytes()...),
		[]byte(strings.Repeat("abcd", 32)+strings.Repeat("1234", 4))) {
		t.Fatalf("expect 16 abcd and 4 1234 but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)
	}
}

func TestZeroRingBuffer(t *testing.T) {
	rb := New(0)
	head, tail := rb.Peek(1)
	if !(head == nil && tail == nil) {
		t.Fatal("expect head and tail are all nil")
	}
	head, tail = rb.PeekAll()
	if !(head == nil && tail == nil) {
		t.Fatal("expect head and tail are all nil")
	}
	if rb.Length() != 0 {
		t.Fatal("expect length is 0")
	}
	if rb.Free() != 0 {
		t.Fatal("expect free is 0")
	}
	buf := []byte(strings.Repeat("1234", 12))
	_, _ = rb.Write(buf)
	if !(rb.Len() == defaultBufferSize && rb.Cap() == defaultBufferSize) {
		t.Fatalf("expect rb.Len()=64 and rb.Cap=64, but got rb.Len()=%d and rb.Cap()=%d", rb.Len(), rb.Cap())
	}
	if !(rb.r == 0 && rb.w == 48 && rb.size == defaultBufferSize && rb.mask == defaultBufferSize-1) {
		t.Fatalf("expect rb.r=0, rb.w=48, rb.size=64, rb.mask=63, but got rb.r=%d, rb.w=%d, rb.size=%d, rb.mask=%d",
			rb.r, rb.w, rb.size, rb.mask)
	}
	if !bytes.Equal(rb.ByteBuffer().Bytes(), buf) {
		t.Fatal("expect it is equal")
	}
	rb.Discard(48)
	if !(rb.IsEmpty() && rb.r == 0 && rb.w == 0) {
		t.Fatalf("expect rb is empty and rb.r=rb.w=0, but got rb.r=%d and rb.w=%d", rb.r, rb.w)
	}
}

func TestRingBuffer_Read(t *testing.T) {
	rb := New(64)

	// check empty or full
	if !rb.IsEmpty() {
		t.Fatalf("expect IsEmpty is true but got false")
	}
	if rb.IsFull() {
		t.Fatalf("expect IsFull is false but got true")
	}
	if rb.Length() != 0 {
		t.Fatalf("expect len 0 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	}
	if rb.Free() != 64 {
		t.Fatalf("expect free 64 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}

	// read empty
	buf := make([]byte, 1024)
	n, err := rb.Read(buf)
	if err == nil {
		t.Fatalf("expect an error but got nil")
	}
	if err != ErrIsEmpty {
		t.Fatalf("expect ErrIsEmpty but got nil")
	}
	if n != 0 {
		t.Fatalf("expect read 0 bytes but got %d", n)
	}
	if rb.Length() != 0 {
		t.Fatalf("expect len 0 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	}
	if rb.Free() != 64 {
		t.Fatalf("expect free 64 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}
	if rb.r != 0 {
		t.Fatalf("expect r.r=0 but got %d. r.w=%d", rb.r, rb.w)
	}

	// write 16 bytes to read
	_, _ = rb.Write([]byte(strings.Repeat("abcd", 4)))
	// read all data from buffer, it will be shrunk from 64 to 32.
	n, err = rb.Read(buf)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if n != 16 {
		t.Fatalf("expect read 16 bytes but got %d", n)
	}
	if rb.Length() != 0 {
		t.Fatalf("expect len 0 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	}
	if rb.Free() != 64 {
		t.Fatalf("expect free 64 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}
	if rb.r != 0 {
		t.Fatalf("expect r.r=0 but got %d. r.w=%d", rb.r, rb.w)
	}

	// write long slice to read, it will scale from 32 to 128 bytes.
	_, _ = rb.Write([]byte(strings.Repeat("abcd", 20)))
	// read all data from buffer, it will be shrunk from 128 to 64.
	n, err = rb.Read(buf)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if n != 80 {
		t.Fatalf("expect read 80 bytes but got %d", n)
	}
	if rb.Length() != 0 {
		t.Fatalf("expect len 0 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	}
	if rb.Free() != 128 {
		t.Fatalf("expect free 128 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}
	if rb.r != 0 {
		t.Fatalf("expect r.r=0 but got %d. r.w=%d", rb.r, rb.w)
	}

	rb.Reset()
	_, _ = rb.Write([]byte(strings.Repeat("1234", 32)))
	if !rb.IsFull() {
		t.Fatal("ring buffer should be full")
	}
	if rb.Free() != 0 {
		t.Fatalf("expect free 0 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}
	if rb.w != 0 {
		t.Fatalf("expect r.2=0 but got %d. r.r=%d", rb.w, rb.r)
	}
	head, tail := rb.Peek(64)
	if !(len(head) == 64 && tail == nil) {
		t.Fatalf("expect len(head)=64 and tail=nil, yet len(head)=%d and tail != nil", len(head))
	}
	if rb.r != 0 {
		t.Fatalf("expect r.r=0 but got %d", rb.r)
	}
	if !bytes.Equal(head, []byte(strings.Repeat("1234", 16))) {
		t.Fatal("should be equal")
	}
	rb.Discard(64)
	if rb.r != 64 {
		t.Fatalf("expect r.r=64 but got %d", rb.r)
	}
	_, _ = rb.Write([]byte(strings.Repeat("1234", 4)))
	if rb.w != 16 {
		t.Fatalf("expect r.w=16 but got %d", rb.w)
	}
	head, tail = rb.Peek(128)
	if !(len(head) == 64 && len(tail) == 16) {
		t.Fatalf("expect len(head)=64 and len(tail)=16, yet len(head)=%d and len(tail)=%d", len(head), len(tail))
	}
	if !(bytes.Equal(head, []byte(strings.Repeat("1234", 16))) && bytes.Equal(tail, []byte(strings.Repeat("1234", 4)))) {
		t.Fatalf("head: %s, tail: %s", string(head), string(tail))
	}

	head, tail = rb.PeekAll()
	if !(len(head) == 64 && len(tail) == 16) {
		t.Fatalf("expect len(head)=64 and len(tail)=16, yet len(head)=%d and len(tail)=%d", len(head), len(tail))
	}
	if !(bytes.Equal(head, []byte(strings.Repeat("1234", 16))) && bytes.Equal(tail, []byte(strings.Repeat("1234", 4)))) {
		t.Fatalf("head: %s, tail: %s", string(head), string(tail))
	}

	rb.Discard(64)
	rb.Discard(16)
	if !rb.isEmpty {
		t.Fatal("should be empty")
	}
}

func TestRingBuffer_ByteInterface(t *testing.T) {
	rb := New(2)

	// write one
	_ = rb.WriteByte('a')
	if rb.Length() != 1 {
		t.Fatalf("expect len 1 byte but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	}
	if rb.Free() != 1 {
		t.Fatalf("expect free 1 byte but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}
	if !bytes.Equal(rb.ByteBuffer().Bytes(), []byte{'a'}) {
		t.Fatalf("expect a but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)
	}
	// check empty or full
	if rb.IsEmpty() {
		t.Fatalf("expect IsEmpty is false but got true")
	}
	if rb.IsFull() {
		t.Fatalf("expect IsFull is false but got true")
	}

	// write two, isFull
	_ = rb.WriteByte('b')
	if rb.Length() != 2 {
		t.Fatalf("expect len 2 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	}
	if rb.Free() != 0 {
		t.Fatalf("expect free 0 byte but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}
	if !bytes.Equal(rb.ByteBuffer().Bytes(), []byte{'a', 'b'}) {
		t.Fatalf("expect a but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)
	}
	// check empty or full
	if rb.IsEmpty() {
		t.Fatalf("expect IsEmpty is false but got true")
	}
	if !rb.IsFull() {
		t.Fatalf("expect IsFull is true but got false")
	}

	// write, it will scale from 2 to 4 bytes.
	_ = rb.WriteByte('c')
	if rb.Length() != 3 {
		t.Fatalf("expect len 3 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	}
	if rb.Free() != 1 {
		t.Fatalf("expect free 1 byte but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}
	if !bytes.Equal(rb.ByteBuffer().Bytes(), []byte{'a', 'b', 'c'}) {
		t.Fatalf("expect a but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)
	}
	// check empty or full
	if rb.IsEmpty() {
		t.Fatalf("expect IsEmpty is false but got true")
	}
	if rb.IsFull() {
		t.Fatalf("expect IsFull is false but got true ")
	}

	// read one
	b, err := rb.ReadByte()
	if err != nil {
		t.Fatalf("ReadByte failed: %v", err)
	}
	if b != 'a' {
		t.Fatalf("expect a but got %c. r.w=%d, r.r=%d", b, rb.w, rb.r)
	}
	if rb.Length() != 2 {
		t.Fatalf("expect len 2 byte but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	}
	if rb.Free() != 2 {
		t.Fatalf("expect free 2 byte but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}
	if !bytes.Equal(rb.ByteBuffer().Bytes(), []byte{'b', 'c'}) {
		t.Fatalf("expect a but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)
	}
	// check empty or full
	if rb.IsEmpty() {
		t.Fatalf("expect IsEmpty is false but got true")
	}
	if rb.IsFull() {
		t.Fatalf("expect IsFull is false but got true")
	}

	// read two
	b, _ = rb.ReadByte()
	if b != 'b' {
		t.Fatalf("expect b but got %c. r.w=%d, r.r=%d", b, rb.w, rb.r)
	}
	if rb.Length() != 1 {
		t.Fatalf("expect len 1 byte but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	}
	if rb.Free() != 3 {
		t.Fatalf("expect free 3 byte but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}
	// check empty or full
	if rb.IsEmpty() {
		t.Fatalf("expect IsEmpty is false but got true ")
	}
	if rb.IsFull() {
		t.Fatalf("expect IsFull is false but got true")
	}

	// read three
	_, _ = rb.ReadByte()
	if rb.Length() != 0 {
		t.Fatalf("expect len 0 byte but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	}
	if rb.Free() != 4 {
		t.Fatalf("expect free 4 byte but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}
	// check empty or full
	if !rb.IsEmpty() {
		t.Fatalf("expect IsEmpty is true but got false")
	}
	if rb.IsFull() {
		t.Fatalf("expect IsFull is false but got true")
	}
}

// func TestShrinkBuffer(t *testing.T) {
// 	testStr := "Hello World!"
// 	testCap := 1024
//
// 	rb := New(testCap)
// 	_, _ = rb.WriteString(testStr)
// 	rb.PeekAll()
// 	rb.Discard(len(testStr))
// 	if rb.Cap() != testCap/2 {
// 		t.Fatalf("expect buffer capacity %d, but got %d", testCap/2, rb.Cap())
// 	}
//
// 	_, _ = rb.WriteString(testStr)
// 	rb.PeekAll()
// 	rb.Reset()
// 	if rb.Cap() != testCap/4 {
// 		t.Fatalf("expect buffer capacity %d, but got %d", testCap/4, rb.Cap())
// 	}
// }
