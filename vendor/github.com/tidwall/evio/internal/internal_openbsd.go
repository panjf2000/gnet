// Copyright 2017 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package internal

// SetKeepAlive sets the keepalive for the connection
func SetKeepAlive(fd, secs int) error {
	// OpenBSD has no user-settable per-socket TCP keepalive options.
	return nil
}
