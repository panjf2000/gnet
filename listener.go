// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package gnet

import (
	"net"
	"os"
	"sync"
)

type listener struct {
	f             *os.File
	fd            int
	ln            net.Listener
	once          sync.Once
	pconn         net.PacketConn
	lnaddr        net.Addr
	addr, network string
}
