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

//go:build linux && poll_opt
// +build linux,poll_opt

package netpoll

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

func epollCtl(epfd int, op int, fd int, event *epollevent) error {
	_, _, errno := unix.RawSyscall6(unix.SYS_EPOLL_CTL, uintptr(epfd), uintptr(op), uintptr(fd), uintptr(unsafe.Pointer(event)), 0, 0)
	if errno != 0 {
		return errnoErr(errno)
	}
	return nil
}
