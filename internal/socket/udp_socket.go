// Copyright (c) 2020 Andy Pan
// Copyright (c) 2017 Max Riveiro
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
	"os"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/errors"
)

func getUDPSockaddr(proto, addr string) (sa unix.Sockaddr, family int, udpAddr *net.UDPAddr, ipv6only bool, err error) {
	var udpVersion string

	udpAddr, err = net.ResolveUDPAddr(proto, addr)
	if err != nil {
		return
	}

	udpVersion, err = determineUDPProto(proto, udpAddr)
	if err != nil {
		return
	}

	switch udpVersion {
	case "udp4":
		sa4 := &unix.SockaddrInet4{Port: udpAddr.Port}

		if udpAddr.IP != nil {
			if len(udpAddr.IP) == 16 {
				copy(sa4.Addr[:], udpAddr.IP[12:16]) // copy last 4 bytes of slice to array
			} else {
				copy(sa4.Addr[:], udpAddr.IP) // copy all bytes of slice to array
			}
		}

		sa, family = sa4, unix.AF_INET
	case "udp6":
		ipv6only = true
		fallthrough
	case "udp":
		sa6 := &unix.SockaddrInet6{Port: udpAddr.Port}

		if udpAddr.IP != nil {
			copy(sa6.Addr[:], udpAddr.IP) // copy all bytes of slice to array
		}

		if udpAddr.Zone != "" {
			var iface *net.Interface
			iface, err = net.InterfaceByName(udpAddr.Zone)
			if err != nil {
				return
			}

			sa6.ZoneId = uint32(iface.Index)
		}

		sa, family = sa6, unix.AF_INET6
	default:
		err = errors.ErrUnsupportedProtocol
	}

	return
}

func determineUDPProto(proto string, addr *net.UDPAddr) (string, error) {
	// If the protocol is set to "udp", we try to determine the actual protocol
	// version from the size of the resolved IP address. Otherwise, we simple use
	// the protocol given to us by the caller.

	if addr.IP.To4() != nil {
		return "udp4", nil
	}

	if addr.IP.To16() != nil {
		return "udp6", nil
	}

	switch proto {
	case "udp", "udp4", "udp6":
		return proto, nil
	}

	return "", errors.ErrUnsupportedUDPProtocol
}

// udpSocket creates an endpoint for communication and returns a file descriptor that refers to that endpoint.
// Argument `reusePort` indicates whether the SO_REUSEPORT flag will be assigned.
func udpSocket(proto, addr string, sockopts ...Option) (fd int, netAddr net.Addr, err error) {
	var (
		family   int
		ipv6only bool
		sockaddr unix.Sockaddr
	)

	if sockaddr, family, netAddr, ipv6only, err = getUDPSockaddr(proto, addr); err != nil {
		return
	}

	if fd, err = sysSocket(family, unix.SOCK_DGRAM, unix.IPPROTO_UDP); err != nil {
		err = os.NewSyscallError("socket", err)
		return
	}
	defer func() {
		if err != nil {
			_ = unix.Close(fd)
		}
	}()

	if family == unix.AF_INET6 && ipv6only {
		if err = SetIPv6Only(fd, 1); err != nil {
			return
		}
	}

	// Allow broadcast.
	if err = os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_BROADCAST, 1)); err != nil {
		return
	}

	for _, sockopt := range sockopts {
		if err = sockopt.SetSockopt(fd, sockopt.Opt); err != nil {
			return
		}
	}

	err = os.NewSyscallError("bind", unix.Bind(fd, sockaddr))

	return
}
