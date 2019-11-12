// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build windows

package gnet

import "os"

func (ln *listener) close() {
	if ln.ln != nil {
		_ = ln.ln.Close()
	}
	if ln.pconn != nil {
		_ = ln.pconn.Close()
	}
	if ln.network == "unix" {
		_ = os.RemoveAll(ln.addr)
	}
}

func (ln *listener) system() error {
	return nil
}
