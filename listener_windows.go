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

package gnet

import (
	"net"
	"os"
	"sync"

	"github.com/panjf2000/gnet/errors"
	"github.com/panjf2000/gnet/internal/netpoll"
	"github.com/panjf2000/gnet/logging"
)

type listener struct {
	once          sync.Once
	ln            net.Listener
	pconn         net.PacketConn
	lnaddr        net.Addr
	addr, network string
}

func (ln *listener) dup() (int, string, error) {
	return netpoll.Dup(0)
}

func (ln *listener) normalize() (err error) {
	switch ln.network {
	case "unix":
		logging.LogErr(os.RemoveAll(ln.addr))
		fallthrough
	case "tcp", "tcp4", "tcp6":
		if ln.ln, err = net.Listen(ln.network, ln.addr); err != nil {
			return
		}
		ln.lnaddr = ln.ln.Addr()
	case "udp", "udp4", "udp6":
		if ln.pconn, err = net.ListenPacket(ln.network, ln.addr); err != nil {
			return
		}
		ln.lnaddr = ln.pconn.LocalAddr()
	default:
		err = errors.ErrUnsupportedProtocol
	}
	return
}

func (ln *listener) close() {
	ln.once.Do(func() {
		if ln.ln != nil {
			logging.LogErr(ln.ln.Close())
		}
		if ln.pconn != nil {
			logging.LogErr(ln.pconn.Close())
		}
		if ln.network == "unix" {
			logging.LogErr(os.RemoveAll(ln.addr))
		}
	})
}

func initListener(network, addr string, _ *Options) (l *listener, err error) {
	l = &listener{network: network, addr: addr}
	err = l.normalize()
	return
}
