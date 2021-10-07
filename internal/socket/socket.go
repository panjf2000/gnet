// Copyright (c) 2020 Andy Pan
// Copyright (c) 2017 Max Riveiro
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

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

// Package socket provides functions that return fd and net.Addr based on
// given the protocol and address with a SO_REUSEPORT option set to the socket.
package socket

import (
	"net"
)

// Option is used for setting an option on socket.
type Option struct {
	SetSockopt func(int, int) error
	Opt        int
}

// TCPSocket calls the internal tcpSocket.
func TCPSocket(proto, addr string, sockopts ...Option) (int, net.Addr, error) {
	return tcpSocket(proto, addr, sockopts...)
}

// UDPSocket calls the internal udpSocket.
func UDPSocket(proto, addr string, sockopts ...Option) (int, net.Addr, error) {
	return udpSocket(proto, addr, sockopts...)
}

// UnixSocket calls the internal udsSocket.
func UnixSocket(proto, addr string, sockopts ...Option) (int, net.Addr, error) {
	return udsSocket(proto, addr, sockopts...)
}
