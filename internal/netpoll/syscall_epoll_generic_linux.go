// Copyright (c) 2021 Andy Pan
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
// SOFTWARE.

// +build linux,!arm64,!riscv64
// +build poll_opt

package netpoll

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

func epollWait(epfd int, events []epollevent, msec int) (int, error) {
	var ep unsafe.Pointer
	if len(events) > 0 {
		ep = unsafe.Pointer(&events[0])
	} else {
		ep = unsafe.Pointer(&zero)
	}
	var (
		np    uintptr
		errno unix.Errno
	)
	if msec == 0 { // non-block system call, use RawSyscall6 to avoid getting preempted by runtime
		np, _, errno = unix.RawSyscall6(unix.SYS_EPOLL_WAIT, uintptr(epfd), uintptr(ep), uintptr(len(events)), 0, 0, 0)
	} else {
		np, _, errno = unix.Syscall6(unix.SYS_EPOLL_WAIT, uintptr(epfd), uintptr(ep), uintptr(len(events)), uintptr(msec), 0, 0)
	}
	if errno != 0 {
		return int(np), errnoErr(errno)
	}
	return int(np), nil
}
