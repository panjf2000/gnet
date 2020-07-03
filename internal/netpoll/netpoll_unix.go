// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2017 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build linux netbsd freebsd dragonfly

package netpoll

import (
	"os"

	"golang.org/x/sys/unix"
)

// SetKeepAlive sets the keepalive for the connection.
func SetKeepAlive(fd, secs int) error {
	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_KEEPALIVE, 1); err != nil {
		return os.NewSyscallError("setsockopt", err)
	}
	if err := unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_KEEPINTVL, secs); err != nil {
		return os.NewSyscallError("setsockopt", err)
	}
	return os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_KEEPIDLE, secs))
}
