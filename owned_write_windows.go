// Copyright (c) 2026 Telego contributors.
// Licensed under the Apache License, Version 2.0. See LICENSE.

package gnet

import (
	"io"
	"net"
)

func (c *conn) WriteOwned(data []byte, release func(error)) (int, error) {
	if release == nil {
		release = func(error) {}
	}
	if _, open := c.loop.connections[c]; !open {
		release(net.ErrClosed)
		return 0, net.ErrClosed
	}
	// The Windows connection writes synchronously and retains no output slice.
	n, err := c.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	release(err)
	return n, err
}
