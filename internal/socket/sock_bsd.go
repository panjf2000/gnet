// Copyright (c) 2020 Andy Pan
// Copyright (c) 2017 Ma Weiwei, Max Riveiro
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build freebsd || dragonfly || darwin
// +build freebsd dragonfly darwin

package socket

import (
	"runtime"

	"golang.org/x/sys/unix"
)

func maxListenerBacklog() int {
	var (
		n   uint32
		err error
	)
	switch runtime.GOOS {
	case "darwin":
		n, err = unix.SysctlUint32("kern.ipc.somaxconn")
	case "freebsd":
		n, err = unix.SysctlUint32("kern.ipc.soacceptqueue")
	}
	if n == 0 || err != nil {
		return unix.SOMAXCONN
	}
	// FreeBSD stores the backlog in a uint16, as does Linux.
	// Assume the other BSDs do too. Truncate number to avoid wrapping.
	// See issue 5030.
	if n > 1<<16-1 {
		n = 1<<16 - 1
	}
	return int(n)
}
