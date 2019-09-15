// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"net"

	"golang.org/x/sys/unix"
)

// SockaddrToTCPOrUnixAddr converts a Sockaddr to a net.TCPAddr or net.UnixAddr.
// Returns nil if conversion fails.
func SockaddrToTCPOrUnixAddr(sa unix.Sockaddr) net.Addr {
	if addr := SockaddrToUnixAddr(sa); addr != nil {
		return addr
	}

	ip, zone := SockaddrToIPAndZone(sa)
	switch sa := sa.(type) {
	case *unix.SockaddrInet4:
		return &net.TCPAddr{IP: ip, Port: sa.Port}
	case *unix.SockaddrInet6:
		return &net.TCPAddr{IP: ip, Port: sa.Port, Zone: zone}
	}
	return nil
}

// SockaddrToUDPAddr converts a Sockaddr to a net.UDPAddr
// Returns nil if conversion fails.
func SockaddrToUDPAddr(sa unix.Sockaddr) *net.UDPAddr {
	ip, zone := SockaddrToIPAndZone(sa)
	switch sa := sa.(type) {
	case *unix.SockaddrInet4:
		return &net.UDPAddr{IP: ip, Port: sa.Port}
	case *unix.SockaddrInet6:
		return &net.UDPAddr{IP: ip, Port: sa.Port, Zone: zone}
	}
	return nil
}

// SockaddrToUnixAddr converts a Sockaddr to a net.UnixAddr
// Returns nil if conversion fails.
func SockaddrToUnixAddr(sa unix.Sockaddr) *net.UnixAddr {
	if s, ok := sa.(*unix.SockaddrUnix); ok {
		return &net.UnixAddr{Name: s.Name, Net: "unix"}
	}
	return nil
}

// IP6ZoneToString converts an IP6 Zone unix int to a net string
// returns "" if zone is 0
func IP6ZoneToString(zone int) string {
	if zone == 0 {
		return ""
	}
	if ifi, err := net.InterfaceByIndex(zone); err == nil {
		return ifi.Name
	}
	return int2decimal(uint(zone))
}

// SockaddrToIPAndZone converts a Sockaddr to a net.IP (with optional IPv6 Zone)
// Returns nil if conversion fails.
func SockaddrToIPAndZone(sa unix.Sockaddr) (net.IP, string) {
	switch sa := sa.(type) {
	case *unix.SockaddrInet4:
		ip := make([]byte, 16)
		// V4InV6Prefix
		ip[10] = 0xff
		ip[11] = 0xff
		copy(ip[12:16], sa.Addr[:])
		return ip, ""

	case *unix.SockaddrInet6:
		ip := make([]byte, 16)
		copy(ip, sa.Addr[:])
		return ip, IP6ZoneToString(int(sa.ZoneId))
	}
	return nil, ""
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
