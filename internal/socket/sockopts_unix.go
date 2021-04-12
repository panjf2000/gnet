// Copyright (c) 2019 Andy Pan
// Copyright (c) 2017 Joshua J Baker
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

// +build linux freebsd dragonfly

package socket

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// SetKeepAlive sets whether the operating system should send
// keep-alive messages on the connection and sets period between keep-alive's.
func SetKeepAlive(fd, secs int) error {
	if secs <= 0 {
		return errors.New("invalid time duration")
	}
	if err := os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_KEEPALIVE, 1)); err != nil {
		return err
	}
	if err := os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_KEEPINTVL, secs)); err != nil {
		return err
	}
	return os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_KEEPIDLE, secs))
}
