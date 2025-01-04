// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (mips || mipsle) && linux && poll_opt
// +build mips mipsle
// +build linux
// +build poll_opt

package netpoll

type epollevent struct {
	events    uint32
	pad_cgo_0 [4]byte
	data      uint64
}
