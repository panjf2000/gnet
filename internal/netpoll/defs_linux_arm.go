// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build poll_opt

package netpoll

type epollevent struct {
	events uint32
	_pad   uint32
	data   [8]byte // to match amd64
}
