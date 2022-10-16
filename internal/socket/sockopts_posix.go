// Copyright (c) 2021 Andy Pan
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
	"os"
	"syscall"

	"github.com/panjf2000/gnet/v2/pkg/errors"
	"golang.org/x/sys/unix"
)

// SetNoDelay controls whether the operating system should delay
// packet transmission in hopes of sending fewer packets (Nagle's algorithm).
//
// The default is true (no delay), meaning that data is
// sent as soon as possible after a Write.
func SetNoDelay(fd, noDelay int) error {
	return os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_NODELAY, noDelay))
}

// SetRecvBuffer sets the size of the operating system's
// receive buffer associated with the connection.
func SetRecvBuffer(fd, size int) error {
	return unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF, size)
}

// SetSendBuffer sets the size of the operating system's
// transmit buffer associated with the connection.
func SetSendBuffer(fd, size int) error {
	return unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_SNDBUF, size)
}

// SetReuseport enables SO_REUSEPORT option on socket.
func SetReuseport(fd, reusePort int) error {
	return os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEPORT, reusePort))
}

// SetReuseAddr enables SO_REUSEADDR option on socket.
func SetReuseAddr(fd, reuseAddr int) error {
	return os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, reuseAddr))
}

// SetIPv6Only restricts a IPv6 socket to only process IPv6 requests or both IPv4 and IPv6 requests.
func SetIPv6Only(fd, ipv6only int) error {
	return unix.SetsockoptInt(fd, unix.IPPROTO_IPV6, unix.IPV6_V6ONLY, ipv6only)
}

// SetLinger sets the behavior of Close on a connection which still
// has data waiting to be sent or to be acknowledged.
//
// If sec < 0 (the default), the operating system finishes sending the
// data in the background.
//
// If sec == 0, the operating system discards any unsent or
// unacknowledged data.
//
// If sec > 0, the data is sent in the background as with sec < 0. On
// some operating systems after sec seconds have elapsed any remaining
// unsent data may be discarded.
func SetLinger(fd, sec int) error {
	var l unix.Linger
	if sec >= 0 {
		l.Onoff = 1
		l.Linger = int32(sec)
	} else {
		l.Onoff = 0
		l.Linger = 0
	}
	return unix.SetsockoptLinger(fd, syscall.SOL_SOCKET, syscall.SO_LINGER, &l)
}

// SetMulticastMembership returns with a socket option function based on the IP
// version. Returns nil when multicast membership cannot be applied.
func SetMulticastMembership(proto, addr string) func(int, int) error {
	var udpVersion string
	udpAddr, err := net.ResolveUDPAddr(proto, addr)
	if err != nil || !udpAddr.IP.IsMulticast() {
		return nil
	}

	udpVersion, err = determineUDPProto(proto, udpAddr)
	if err != nil {
		return nil
	}

	switch udpVersion {
	case "udp4":
		return func(fd int, ifIndex int) error {
			return SetIPv4MulticastMembership(fd, udpAddr.IP, ifIndex)
		}
	case "udp6":
		return func(fd int, ifIndex int) error {
			return SetIPv6MulticastMembership(fd, udpAddr.IP, ifIndex)
		}
	default:
		return nil
	}
}

// SetIPv4MulticastMemership joins fd to the specified multicast IPv4 address.
// ifIndex is the index of the interface where the multicast datagrams will be
// received. If ifIndex is 0 then the operating system will choose the default,
// it is usually needed when the host has multiple network interfaces configured.
func SetIPv4MulticastMembership(fd int, mcast net.IP, ifIndex int) error {
	// Multicast interfaces are selected by IP address on IPv4 (and by index on IPv6)
	ip, err := interfaceFirstIPv4Addr(ifIndex)
	if err != nil {
		return err
	}

	mreq := &unix.IPMreq{}
	copy(mreq.Multiaddr[:], mcast.To4())
	copy(mreq.Interface[:], ip.To4())

	if ifIndex > 0 {
		if err := os.NewSyscallError("setsockopt", unix.SetsockoptInet4Addr(fd, syscall.IPPROTO_IP, syscall.IP_MULTICAST_IF, mreq.Interface)); err != nil {
			return err
		}
	}

	if err := os.NewSyscallError("setsockopt", unix.SetsockoptByte(fd, syscall.IPPROTO_IP, syscall.IP_MULTICAST_LOOP, 0)); err != nil {
		return err
	}
	return os.NewSyscallError("setsockopt", unix.SetsockoptIPMreq(fd, syscall.IPPROTO_IP, syscall.IP_ADD_MEMBERSHIP, mreq))
}

// SetIPv6MulticastMemership joins fd to the specified multicast IPv6 address.
// ifIndex is the index of the interface where the multicast datagrams will be
// received. If ifIndex is 0 then the operating system will choose the default,
// it is usually needed when the host has multiple network interfaces configured.
func SetIPv6MulticastMembership(fd int, mcast net.IP, ifIndex int) error {
	mreq := &unix.IPv6Mreq{}
	mreq.Interface = uint32(ifIndex)
	copy(mreq.Multiaddr[:], mcast.To16())

	if ifIndex > 0 {
		if err := os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, syscall.IPPROTO_IPV6, syscall.IPV6_MULTICAST_IF, ifIndex)); err != nil {
			return err
		}
	}

	if err := os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, syscall.IPPROTO_IPV6, syscall.IPV6_MULTICAST_LOOP, 0)); err != nil {
		return err
	}
	return os.NewSyscallError("setsockopt", unix.SetsockoptIPv6Mreq(fd, syscall.IPPROTO_IPV6, syscall.IPV6_JOIN_GROUP, mreq))
}

// interfaceFirstIPv4Addr returns the first IPv4 address of the interface.
func interfaceFirstIPv4Addr(ifIndex int) (net.IP, error) {
	if ifIndex == 0 {
		return net.IP([]byte{0, 0, 0, 0}), nil
	}
	iface, err := net.InterfaceByIndex(ifIndex)
	if err != nil {
		return nil, err
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			return nil, err
		}
		if ip.To4() != nil {
			return ip, nil
		}
	}
	return nil, errors.ErrNoIPv4AddressOnInterface
}
