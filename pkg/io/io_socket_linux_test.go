// Copyright (c) 2025 The Gnet Authors. All rights reserved.
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

//go:build linux

package io

import (
	"bytes"
	"testing"

	"golang.org/x/sys/unix"
)

func TestReadWriteSocketNB(t *testing.T) {
	a, b := nonblockingPair(t)
	buf := make([]byte, 32)

	if n, err := ReadSocketNB(b, buf); n != -1 || err != unix.EAGAIN {
		t.Fatalf("empty socket: n=%d err=%v, want -1 / EAGAIN", n, err)
	}

	// Empty reads and writes retain read/write semantics and do not consume data.
	if n, err := WriteSocketNB(a, nil); n != 0 || err != nil {
		t.Fatalf("zero-length write: n=%d err=%v, want 0 / nil", n, err)
	}
	msg := []byte("socket-specific raw syscall")
	if n, err := WriteSocketNB(a, msg); n != len(msg) || err != nil {
		t.Fatalf("write: n=%d err=%v, want %d / nil", n, err, len(msg))
	}
	if n, err := ReadSocketNB(b, nil); n != 0 || err != nil {
		t.Fatalf("zero-length read: n=%d err=%v, want 0 / nil", n, err)
	}
	if n, err := ReadSocketNB(b, buf); n != len(msg) || err != nil || !bytes.Equal(buf[:n], msg) {
		t.Fatalf("read: n=%d err=%v data=%q", n, err, buf[:max(n, 0)])
	}

	// A non-empty read after the peer closes must report EOF as 0 / nil.
	if err := unix.Close(a); err != nil {
		t.Fatal(err)
	}
	if n, err := ReadSocketNB(b, buf); n != 0 || err != nil {
		t.Fatalf("peer close: n=%d err=%v, want 0 / nil", n, err)
	}
}

func TestReadWriteSocketNBErrors(t *testing.T) {
	if n, err := ReadSocketNB(-1, []byte("x")); n != -1 || err != unix.EBADF {
		t.Fatalf("bad read fd: n=%d err=%v, want -1 / EBADF", n, err)
	}
	if n, err := WriteSocketNB(-1, []byte("x")); n != -1 || err != unix.EBADF {
		t.Fatalf("bad write fd: n=%d err=%v, want -1 / EBADF", n, err)
	}

	a, b := nonblockingPair(t)
	if err := unix.Close(b); err != nil {
		t.Fatal(err)
	}
	if n, err := WriteSocketNB(a, []byte("x")); n != -1 || err != unix.EPIPE {
		t.Fatalf("write after peer close: n=%d err=%v, want -1 / EPIPE", n, err)
	}
}

func TestWriteSocketNBFullBufferDoesNotBlock(t *testing.T) {
	a, _ := nonblockingPair(t)
	chunk := make([]byte, 64*1024)
	total := 0
	for i := 0; i < 1<<14; i++ {
		n, err := WriteSocketNB(a, chunk)
		if err == unix.EAGAIN {
			if total == 0 {
				t.Fatal("EAGAIN before any byte was accepted")
			}
			return
		}
		if err != nil {
			t.Fatalf("unexpected error after %d bytes: %v", total, err)
		}
		total += n
	}
	t.Fatalf("socket buffer never filled up after %d bytes", total)
}

func TestReadSocketNBNoAllocOnEAGAIN(t *testing.T) {
	_, b := nonblockingPair(t)
	buf := make([]byte, 64)
	if allocs := testing.AllocsPerRun(200, func() { _, _ = ReadSocketNB(b, buf) }); allocs != 0 {
		t.Fatalf("ReadSocketNB allocates %.1f times per EAGAIN call", allocs)
	}
}

func TestWriteSocketNBNoAllocOnEAGAIN(t *testing.T) {
	a, _ := nonblockingPair(t)
	chunk := make([]byte, 64*1024)
	for {
		_, err := WriteSocketNB(a, chunk)
		if err == unix.EAGAIN {
			break
		}
		if err != nil {
			t.Fatalf("fill socket buffer: %v", err)
		}
	}
	if allocs := testing.AllocsPerRun(200, func() { _, _ = WriteSocketNB(a, chunk) }); allocs != 0 {
		t.Fatalf("WriteSocketNB allocates %.1f times per EAGAIN call", allocs)
	}
}
