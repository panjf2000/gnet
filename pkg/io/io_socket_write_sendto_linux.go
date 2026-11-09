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

//go:build linux && gnet_sendto

package io

import (
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

// WriteSocketNB uses sendto(2) with MSG_NOSIGNAL for non-blocking sockets.
// A NULL destination and zero address length make this equivalent to send(2)
// on the already-connected sockets used by the event loop. MSG_NOSIGNAL
// suppresses SIGPIPE for this syscall while preserving EPIPE as the returned
// error. The gnet_sendto build tag enables this path for isolated benchmarking;
// the default build retains write(2).
//
// It must only be used with socket descriptors. Eventfd and other non-socket
// descriptors must continue to use WriteNB.
func WriteSocketNB(fd int, p []byte) (n int, err error) {
	ptr := unsafe.Pointer(&_zero)
	if len(p) > 0 {
		ptr = unsafe.Pointer(&p[0])
	}

	r, _, e := unix.RawSyscall6(
		unix.SYS_SENDTO,
		uintptr(fd), uintptr(ptr), uintptr(len(p)), uintptr(unix.MSG_NOSIGNAL), 0, 0,
	)
	runtime.KeepAlive(p)
	if e != 0 {
		return -1, errnoErr(e)
	}
	return int(r), nil
}
