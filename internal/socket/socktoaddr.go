// Copyright (c) 2019 Andy Pan
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// +build linux freebsd dragonfly darwin

package socket

import (
	"net"

	"golang.org/x/sys/unix"
)

// SockaddrToTCPOrUnixAddr converts a Sockaddr to a net.TCPAddr or net.UnixAddr.
// Returns nil if conversion fails.
func SockaddrToTCPOrUnixAddr(sa unix.Sockaddr) net.Addr {
	switch sa := sa.(type) {
	case *unix.SockaddrInet4:
		ip := sockaddrInet4ToIP(sa)
		return &net.TCPAddr{IP: ip, Port: sa.Port}
	case *unix.SockaddrInet6:
		ip, zone := sockaddrInet6ToIPAndZone(sa)
		return &net.TCPAddr{IP: ip, Port: sa.Port, Zone: zone}
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
		ip := sockaddrInet4ToIP(sa)
		return &net.UDPAddr{IP: ip, Port: sa.Port}
	case *unix.SockaddrInet6:
		ip, zone := sockaddrInet6ToIPAndZone(sa)
		return &net.UDPAddr{IP: ip, Port: sa.Port, Zone: zone}
	}
	return nil
}

// sockaddrInet4ToIPAndZone converts a SockaddrInet4 to a net.IP.
// It returns nil if conversion fails.
func sockaddrInet4ToIP(sa *unix.SockaddrInet4) net.IP {
	ip := make([]byte, 16)
	// V4InV6Prefix
	ip[10] = 0xff
	ip[11] = 0xff
	copy(ip[12:16], sa.Addr[:])
	return ip
}

// sockaddrInet6ToIPAndZone converts a SockaddrInet6 to a net.IP with IPv6 Zone.
// It returns nil if conversion fails.
func sockaddrInet6ToIPAndZone(sa *unix.SockaddrInet6) (net.IP, string) {
	ip := make([]byte, 16)
	copy(ip, sa.Addr[:])
	return ip, ip6ZoneToString(int(sa.ZoneId))
}

// ip6ZoneToString converts an IP6 Zone unix int to a net string
// returns "" if zone is 0.
func ip6ZoneToString(zone int) string {
	if zone == 0 {
		return ""
	}
	if ifi, err := net.InterfaceByIndex(zone); err == nil {
		return ifi.Name
	}
	return int2decimal(uint(zone))
}

// Convert int to decimal string.
func int2decimal(i uint) string {
	if i == 0 {
		return "0"
	}

	// Assemble decimal in reverse order.
	var b [32]byte
	bp := len(b)
	for ; i > 0; i /= 10 {
		bp--
		b[bp] = byte(i%10) + '0'
	}
	return string(b[bp:])
}
