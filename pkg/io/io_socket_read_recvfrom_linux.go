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

//go:build linux && !gnet_legacy_read

package io

import (
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

// ReadSocketNB uses recvfrom(2) for non-blocking sockets. The NULL source
// address and address-length arguments make this equivalent to recv(2) for the
// stream sockets used by the event loop. This is the default Linux socket-read
// path; gnet_legacy_read restores the previous read(2) implementation.
//
// It must only be used with non-blocking socket descriptors. Eventfd and other
// non-socket descriptors must continue to use ReadNB.
func ReadSocketNB(fd int, p []byte) (n int, err error) {
	ptr := unsafe.Pointer(&_zero)
	if len(p) > 0 {
		ptr = unsafe.Pointer(&p[0])
	}

	r, _, e := unix.RawSyscall6(
		unix.SYS_RECVFROM,
		uintptr(fd), uintptr(ptr), uintptr(len(p)), 0, 0, 0,
	)
	runtime.KeepAlive(p)
	if e != 0 {
		return -1, errnoErr(e)
	}
	return int(r), nil
}
