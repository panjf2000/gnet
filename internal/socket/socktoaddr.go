// Copyright (c) 2019 The Gnet Authors. All rights reserved.
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

package socket

import (
	"net"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/internal/bs"
	bsPool "github.com/panjf2000/gnet/v2/pkg/pool/byteslice"
)

// SockaddrToTCPOrUnixAddr converts a Sockaddr to a net.TCPAddr or net.UnixAddr.
// Returns nil if conversion fails.
func SockaddrToTCPOrUnixAddr(sa unix.Sockaddr) net.Addr {
	switch sa := sa.(type) {
	case *unix.SockaddrInet4:
		return &net.TCPAddr{IP: sa.Addr[0:], Port: sa.Port}
	case *unix.SockaddrInet6:
		return &net.TCPAddr{IP: sa.Addr[0:], Port: sa.Port, Zone: ip6ZoneToString(sa.ZoneId)}
	case *unix.SockaddrUnix:
		return &net.UnixAddr{Name: sa.Name, Net: "unix"}
	}
	return nil
}

// SockaddrToUDPAddr converts a Sockaddr to a net.UDPAddr
// Returns nil if conversion fails.
func SockaddrToUDPAddr(sa unix.Sockaddr) net.Addr {
	switch sa := sa.(type) {
	case *unix.SockaddrInet4:
		return &net.UDPAddr{IP: sa.Addr[0:], Port: sa.Port}
	case *unix.SockaddrInet6:
		return &net.UDPAddr{IP: sa.Addr[0:], Port: sa.Port, Zone: ip6ZoneToString(sa.ZoneId)}
	}
	return nil
}

// ip6ZoneToString converts an IP6 Zone unix int to a net string,
// returns "" if zone is 0.
func ip6ZoneToString(zone uint32) string {
	if zone == 0 {
		return ""
	}
	if ifi, err := net.InterfaceByIndex(int(zone)); err == nil {
		return ifi.Name
	}
	return uint2decimalStr(uint(zone))
}

// uint2decimalStr converts val to a decimal string.
func uint2decimalStr(val uint) string {
	if val == 0 { // avoid string allocation
		return "0"
	}
	buf := bsPool.Get(20) // big enough for 64bit value base 10
	i := len(buf) - 1
	for val >= 10 {
		q := val / 10
		buf[i] = byte('0' + val - q*10)
		i--
		val = q
	}
	// val < 10
	buf[i] = byte('0' + val)
	return bs.BytesToString(buf[i:])
}
