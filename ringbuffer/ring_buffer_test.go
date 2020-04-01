// Copyright 2019 smallnest&panjf200. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

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
	if bytes.Compare(rb.ByteBuffer().Bytes(), []byte(strings.Repeat("abcd", 4))) != 0 {
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
	if bytes.Compare(rb.ByteBuffer().Bytes(), []byte(strings.Repeat("abcd", 16))) != 0 {
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

	if bytes.Compare(rb.ByteBuffer().Bytes(), []byte(strings.Repeat("abcd", 20))) != 0 {
		t.Fatalf("expect 16 abcd but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)
	}

	rb.Reset()
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
	rb.Read(buf)
	if rb.Length() != 3 {
		t.Fatalf("expect len 3 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	}
	_, _ = rb.Write([]byte(strings.Repeat("abcd", 15)))

	if bytes.Compare(rb.ByteBuffer().Bytes(), []byte("bcd"+strings.Repeat("abcd", 15))) != 0 {
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
	rb.Read(buf)
	n, _ = rb.Write([]byte(strings.Repeat("1234", 4)))
	if n != 16 {
		t.Fatalf("expect write 16 bytes but got %d", n)
	}
	if rb.Free() != 0 {
		t.Fatalf("expect free 0 bytes but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}
	if !bytes.Equal(append(buf, rb.ByteBuffer().Bytes()...), []byte(strings.Repeat("abcd", 32)+strings.Repeat("1234", 4))) {
		t.Fatalf("expect 16 abcd and 4 1234 but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)
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
	rb.Write([]byte(strings.Repeat("abcd", 4)))
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
	if rb.r != 16 {
		t.Fatalf("expect r.r=16 but got %d. r.w=%d", rb.r, rb.w)
	}

	// write long slice to read, it will scale from 64 to 128 bytes.
	rb.Write([]byte(strings.Repeat("abcd", 20)))
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
	if rb.r != 80 {
		t.Fatalf("expect r.r=80 but got %d. r.w=%d", rb.r, rb.w)
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
	if bytes.Compare(rb.ByteBuffer().Bytes(), []byte{'a'}) != 0 {
		t.Fatalf("expect a but got %s. r.w=%d, r.r=%d", rb.ByteBuffer().Bytes(), rb.w, rb.r)
	}
	// check empty or full
	if rb.IsEmpty() {
		t.Fatalf("expect IsEmpty is false but got true")
	}
	if rb.IsFull() {
		t.Fatalf("expect IsFull is false but got true")
	}

	// write to, isFull
	_ = rb.WriteByte('b')
	if rb.Length() != 2 {
		t.Fatalf("expect len 2 bytes but got %d. r.w=%d, r.r=%d", rb.Length(), rb.w, rb.r)
	}
	if rb.Free() != 0 {
		t.Fatalf("expect free 0 byte but got %d. r.w=%d, r.r=%d", rb.Free(), rb.w, rb.r)
	}
	if bytes.Compare(rb.ByteBuffer().Bytes(), []byte{'a', 'b'}) != 0 {
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
	if bytes.Compare(rb.ByteBuffer().Bytes(), []byte{'a', 'b', 'c'}) != 0 {
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
	if bytes.Compare(rb.ByteBuffer().Bytes(), []byte{'b', 'c'}) != 0 {
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
	b, _ = rb.ReadByte()
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
