// Copyright (c) 2023 Andy Pan.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build windows
// +build windows

package gnet

import (
	"net"
	"syscall"
)

func SysClose(fd int) error {
	return syscall.CloseHandle(syscall.Handle(fd))
}

func NetDial(network, addr string) (net.Conn, error) {
	if network == "unix" {
		laddr, _ := net.ResolveUnixAddr(network, unixAddr(addr))
		raddr, _ := net.ResolveUnixAddr(network, addr)
		return net.DialUnix(network, laddr, raddr)
	}
	return net.Dial(network, addr)
}
