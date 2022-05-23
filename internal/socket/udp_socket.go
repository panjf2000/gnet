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

package socket

import (
	"net"
	"os"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/pkg/errors"
)

// GetUDPSockAddr the structured addresses based on the protocol and raw address.
func GetUDPSockAddr(proto, addr string) (sa unix.Sockaddr, family int, udpAddr *net.UDPAddr, ipv6only bool, err error) {
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
func udpSocket(proto, addr string, connect bool, sockOpts ...Option) (fd int, netAddr net.Addr, err error) {
	var (
		family   int
		ipv6only bool
		sa       unix.Sockaddr
	)

	if sa, family, netAddr, ipv6only, err = GetUDPSockAddr(proto, addr); err != nil {
		return
	}

	if fd, err = sysSocket(family, unix.SOCK_DGRAM, unix.IPPROTO_UDP); err != nil {
		err = os.NewSyscallError("socket", err)
		return
	}
	defer func() {
		// ignore EINPROGRESS for non-blocking socket connect, should be processed by caller
		if err != nil {
			if err, ok := err.(*os.SyscallError); ok && err.Err == unix.EINPROGRESS {
				return
			}
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

	for _, sockOpt := range sockOpts {
		if err = sockOpt.SetSockOpt(fd, sockOpt.Opt); err != nil {
			return
		}
	}

	if connect {
		err = os.NewSyscallError("connect", unix.Connect(fd, sa))
	} else {
		err = os.NewSyscallError("bind", unix.Bind(fd, sa))
	}

	return
}
