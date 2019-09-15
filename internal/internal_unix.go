// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2017 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build netbsd freebsd dragonfly linux

package internal

import (
	"golang.org/x/sys/unix"
)

// SetKeepAlive ...
func SetKeepAlive(fd, secs int) error {
	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_KEEPALIVE, 1); err != nil {
		return err
	}
	if err := unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_KEEPINTVL, secs); err != nil {
		return err
	}
	return unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_KEEPIDLE, secs)
}
