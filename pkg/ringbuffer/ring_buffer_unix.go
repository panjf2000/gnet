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

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package ringbuffer

import "golang.org/x/sys/unix"

// ========================= gnet specific APIs =========================

// CopyFromSocket copies data from a socket fd into ring-buffer.
func (rb *RingBuffer) CopyFromSocket(fd int) (n int, err error) {
	n, err = unix.Read(fd, rb.buf[rb.w:])
	if n > 0 {
		rb.isEmpty = false
		rb.w += n
		if rb.w == rb.size {
			rb.w = 0
		}
	}
	return
}

// Rewind moves the data from its tail to head and rewind its pointers of read and write.
func (rb *RingBuffer) Rewind() int {
	if rb.IsEmpty() {
		rb.Reset()
		return 0
	}
	if rb.w != 0 {
		return 0
	}
	if rb.r < rb.size-rb.r {
		rb.grow(rb.size + rb.size/2)
		return rb.size - rb.r
	}
	n := copy(rb.buf, rb.buf[rb.r:])
	rb.r = 0
	rb.w = n
	return n
}
