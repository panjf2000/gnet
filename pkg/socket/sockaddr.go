// Copyright (c) 2019 The Gnet Authors. All rights reserved.
// Copyright (c) 2012 The Go Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//     https://github.com/libp2p/go-sockaddr?tab=BSD-3-Clause-1-ov-file#readme
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package socket

import (
	"net"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/pkg/bs"
	bsPool "github.com/panjf2000/gnet/v2/pkg/pool/byteslice"
)

// NetAddrToSockaddr converts a net.Addr to a Sockaddr.
// Returns nil if the input is invalid or conversion is not possible.
func NetAddrToSockaddr(addr net.Addr) unix.Sockaddr {
	switch addr := addr.(type) {
	case *net.IPAddr:
		return IPAddrToSockaddr(addr)
	case *net.TCPAddr:
		return TCPAddrToSockaddr(addr)
	case *net.UDPAddr:
		return UDPAddrToSockaddr(addr)
	case *net.UnixAddr:
		sa, _ := UnixAddrToSockaddr(addr)
		return sa
	default:
		return nil
	}
}

// IPAddrToSockaddr converts a net.IPAddr to a Sockaddr.
// Returns nil if conversion fails.
func IPAddrToSockaddr(addr *net.IPAddr) unix.Sockaddr {
	return IPToSockaddr(addr.IP, 0, addr.Zone)
}

// TCPAddrToSockaddr converts a net.TCPAddr to a Sockaddr.
// Returns nil if conversion fails.
func TCPAddrToSockaddr(addr *net.TCPAddr) unix.Sockaddr {
	return IPToSockaddr(addr.IP, addr.Port, addr.Zone)
}

// UDPAddrToSockaddr converts a net.UDPAddr to a Sockaddr.
// Returns nil if conversion fails.
func UDPAddrToSockaddr(addr *net.UDPAddr) unix.Sockaddr {
	return IPToSockaddr(addr.IP, addr.Port, addr.Zone)
}

// IPToSockaddr converts a net.IP (with optional IPv6 Zone) to a Sockaddr
// Returns nil if conversion fails.
func IPToSockaddr(ip net.IP, port int, zone string) unix.Sockaddr {
	// Unspecified?
	if ip == nil {
		if zone != "" {
			return &unix.SockaddrInet6{Port: port, ZoneId: uint32(ip6ZoneToInt(zone))}
		}
		return &unix.SockaddrInet4{Port: port}
	}

	// Valid IPv4?
	if ip4 := ip.To4(); ip4 != nil && zone == "" {
		sa := unix.SockaddrInet4{Port: port}
		copy(sa.Addr[:], ip4) // last 4 bytes
		return &sa
	}

	// Valid IPv6 address?
	if ip6 := ip.To16(); ip6 != nil {
		sa := unix.SockaddrInet6{Port: port, ZoneId: uint32(ip6ZoneToInt(zone))}
		copy(sa.Addr[:], ip6)
		return &sa
	}

	return nil
}

// UnixAddrToSockaddr converts a net.UnixAddr to a Sockaddr, and returns
// the type (unix.SOCK_STREAM, unix.SOCK_DGRAM, unix.SOCK_SEQPACKET)
// Returns (nil, 0) if conversion fails.
func UnixAddrToSockaddr(addr *net.UnixAddr) (unix.Sockaddr, int) {
	t := 0
	switch addr.Net {
	case "unix":
		t = unix.SOCK_STREAM
	case "unixgram":
		t = unix.SOCK_DGRAM
	case "unixpacket":
		t = unix.SOCK_SEQPACKET
	default:
		return nil, 0
	}
	return &unix.SockaddrUnix{Name: addr.Name}, t
}

// SockaddrToTCPOrUnixAddr converts a unix.Sockaddr to a net.TCPAddr or net.UnixAddr.
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

// SockaddrToUDPAddr converts a unix.Sockaddr to a net.UDPAddr
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

// ip6ZoneToInt converts an IP6 Zone net string to a unix int.
// Returns 0 if zone is "".
func ip6ZoneToInt(zone string) int {
	if zone == "" {
		return 0
	}
	if ifi, err := net.InterfaceByName(zone); err == nil {
		return ifi.Index
	}
	n, _, _ := dtoi(zone, 0)
	return n
}

// ip6ZoneToString converts an IP6 Zone unix int to a net string,
// Returns "" if zone is 0.
func ip6ZoneToString(zone uint32) string {
	if zone == 0 {
		return ""
	}
	if ifi, err := net.InterfaceByIndex(int(zone)); err == nil {
		return ifi.Name
	}
	return itod(uint(zone))
}

// itod converts uint to a decimal string.
func itod(v uint) string {
	if v == 0 { // avoid string allocation
		return "0"
	}
	// Assemble decimal in reverse order.
	buf := bsPool.Get(32)
	i := len(buf) - 1
	for ; v > 0; v /= 10 {
		buf[i] = byte(v%10 + '0')
		i--
	}
	return bs.BytesToString(buf[i:])
}

// Bigger than we need, not too big to worry about overflow.
const big = 0xFFFFFF

// Decimal to integer starting at &s[i0].
// Returns number, new offset, success.
func dtoi(s string, i0 int) (n int, i int, ok bool) {
	n = 0
	for i = i0; i < len(s) && '0' <= s[i] && s[i] <= '9'; i++ {
		n = n*10 + int(s[i]-'0')
		if n >= big {
			return 0, i, false
		}
	}
	if i == i0 {
		return 0, i, false
	}
	return n, i, true
}
