// +build darwin netbsd freebsd openbsd dragonfly linux

package netpoll

import (
	"net"

	"github.com/libp2p/go-reuseport"
)

// ReusePortListenPacket returns a net.PacketConn for UDP.
func ReusePortListenPacket(proto, addr string) (net.PacketConn, error) {
	return reuseport.ListenPacket(proto, addr)
}

// ReusePortListen returns a net.Listener for TCP.
func ReusePortListen(proto, addr string) (net.Listener, error) {
	return reuseport.Listen(proto, addr)
}
