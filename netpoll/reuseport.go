// +build darwin netbsd freebsd openbsd dragonfly linux

package netpoll

import (
	"net"

	"github.com/libp2p/go-reuseport"
)

func ReusePortListenPacket(proto, addr string) (l net.PacketConn, err error) {
	return reuseport.ListenPacket(proto, addr)
}

func ReusePortListen(proto, addr string) (l net.Listener, err error) {
	return reuseport.Listen(proto, addr)
}
