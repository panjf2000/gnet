// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2017 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package internal

import "golang.org/x/sys/unix"

// SetKeepAlive sets the keepalive for the connection
func SetKeepAlive(fd, secs int) error {
	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, 0x8, 1); err != nil {
		return err
	}
	switch err := unix.SetsockoptInt(fd, unix.IPPROTO_TCP, 0x101, secs); err {
	case nil, unix.ENOPROTOOPT: // OS X 10.7 and earlier don't support this option
	default:
		return err
	}
	return unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_KEEPALIVE, secs)
}
