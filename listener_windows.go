// Copyright (c) 2019 Andy Pan
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
