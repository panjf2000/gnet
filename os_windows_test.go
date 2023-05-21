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
