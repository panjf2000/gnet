// +build linux darwin dragonfly freebsd netbsd openbsd

// Copyright (C) 2017 Max Riveiro
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package reuseport

import (
	"errors"
	"net"
	"os"
	"syscall"
)

var (
	listenerBacklogMaxSize    = maxListenerBacklog()
	errUnsupportedTCPProtocol = errors.New("only tcp, tcp4, tcp6 are supported")
)

func getTCPSockaddr(proto, addr string) (sa syscall.Sockaddr, soType int, err error) {
	var tcp *net.TCPAddr

	tcp, err = net.ResolveTCPAddr(proto, addr)
	if err != nil && tcp.IP != nil {
		return nil, -1, err
	}

	tcpVersion, err := determineTCPProto(proto, tcp)
	if err != nil {
		return nil, -1, err
	}

	switch tcpVersion {
	case "tcp":
		return &syscall.SockaddrInet4{Port: tcp.Port}, syscall.AF_INET, nil
	case "tcp4":
		sa := &syscall.SockaddrInet4{Port: tcp.Port}

		if tcp.IP != nil {
			copy(sa.Addr[:], tcp.IP[12:16]) // copy last 4 bytes of slice to array
		}

		return sa, syscall.AF_INET, nil
	case "tcp6":
		sa := &syscall.SockaddrInet6{Port: tcp.Port}

		if tcp.IP != nil {
			copy(sa.Addr[:], tcp.IP) // copy all bytes of slice to array
		}

		if tcp.Zone != "" {
			iface, err := net.InterfaceByName(tcp.Zone)
			if err != nil {
				return nil, -1, err
			}

			sa.ZoneId = uint32(iface.Index)
		}

		return sa, syscall.AF_INET6, nil
	}

	return nil, -1, errUnsupportedProtocol
}

func determineTCPProto(proto string, ip *net.TCPAddr) (string, error) {
	// If the protocol is set to "tcp", we try to determine the actual protocol
	// version from the size of the resolved IP address. Otherwise, we simple use
	// the protcol given to us by the caller.

	if ip.IP.To4() != nil {
		return "tcp4", nil
	}

	if ip.IP.To16() != nil {
		return "tcp6", nil
	}

	switch proto {
	case "tcp", "tcp4", "tcp6":
		return proto, nil
	}

	return "", errUnsupportedTCPProtocol
}

// NewReusablePortListener returns net.FileListener that created from
// a file discriptor for a socket with SO_REUSEPORT option.
func NewReusablePortListener(proto, addr string) (l net.Listener, err error) {
	var (
		soType, fd int
		file       *os.File
		sockaddr   syscall.Sockaddr
	)

	if sockaddr, soType, err = getSockaddr(proto, addr); err != nil {
		return nil, err
	}

	syscall.ForkLock.RLock()
	if fd, err = syscall.Socket(soType, syscall.SOCK_STREAM, syscall.IPPROTO_TCP); err != nil {
		syscall.ForkLock.RUnlock()

		return nil, err
	}
	syscall.ForkLock.RUnlock()

	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		syscall.Close(fd)
		return nil, err
	}

	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, reusePort, 1); err != nil {
		syscall.Close(fd)
		return nil, err
	}

	if err = syscall.Bind(fd, sockaddr); err != nil {
		syscall.Close(fd)
		return nil, err
	}

	// Set backlog size to the maximum
	if err = syscall.Listen(fd, listenerBacklogMaxSize); err != nil {
		syscall.Close(fd)
		return nil, err
	}

	file = os.NewFile(uintptr(fd), getSocketFileName(proto, addr))
	if l, err = net.FileListener(file); err != nil {
		file.Close()
		return nil, err
	}

	if err = file.Close(); err != nil {
		return nil, err
	}

	return l, err
}
