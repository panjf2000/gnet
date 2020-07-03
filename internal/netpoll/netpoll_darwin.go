// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2017 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

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
	switch err := unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_KEEPINTVL, secs); err {
	case nil, unix.ENOPROTOOPT: // OS X 10.7 and earlier don't support this option
	default:
		return os.NewSyscallError("setsockopt", err)
	}
	return os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_KEEPALIVE, secs))
}
