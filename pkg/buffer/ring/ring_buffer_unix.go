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

package ring

import (
	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/internal/io"
)

// ========================= gnet specific APIs =========================

// CopyFromSocket copies data from a socket fd into ring-buffer.
func (rb *Buffer) CopyFromSocket(fd int) (n int, err error) {
	if rb.r == rb.w {
		if !rb.isEmpty {
			rb.grow(rb.size + rb.size/2)
			n, err = unix.Read(fd, rb.buf[rb.w:])
			if n > 0 {
				rb.w = (rb.w + n) % rb.size
			}
			return
		}
		rb.r, rb.w = 0, 0
		n, err = unix.Read(fd, rb.buf)
		if n > 0 {
			rb.w = (rb.w + n) % rb.size
			rb.isEmpty = false
		}
		return
	}
	if rb.w < rb.r {
		n, err = unix.Read(fd, rb.buf[rb.w:rb.r])
		if n > 0 {
			rb.w = (rb.w + n) % rb.size
		}
		return
	}
	rb.bs[0] = rb.buf[rb.w:]
	rb.bs[1] = rb.buf[:rb.r]
	n, err = io.Readv(fd, rb.bs)
	if n > 0 {
		rb.w = (rb.w + n) % rb.size
	}
	return
}

// Rewind moves the data from its tail to head and rewind its pointers of read and write.
func (rb *Buffer) Rewind() (n int) {
	if rb.IsEmpty() {
		rb.Reset()
		return
	}
	if rb.w == 0 {
		if rb.r < rb.size-rb.r {
			rb.grow(rb.size + rb.size - rb.r)
			return rb.size - rb.r
		}
		n = copy(rb.buf, rb.buf[rb.r:])
		rb.r = 0
		rb.w = n
	} else if rb.size-rb.w < DefaultBufferSize {
		if rb.r < rb.w-rb.r {
			rb.grow(rb.size + rb.w - rb.r)
			return rb.w - rb.r
		}
		n = copy(rb.buf, rb.buf[rb.r:rb.w])
		rb.r = 0
		rb.w = n
	}
	return
}
