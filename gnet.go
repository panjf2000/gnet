// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package gnet

import (
	"errors"
	"net"
	"os"
	"strings"
	"time"
)

var errClosing = errors.New("closing")

// Action is an action that occurs after the completion of an event.
type Action int

const (
	// None indicates that no action should occur following an event.
	None Action = iota
	// Close closes the connection.
	Close
	// Shutdown shutdowns the server.
	Shutdown
)

// Options are set when the client opens.
type Options struct {
	// TCPKeepAlive (SO_KEEPALIVE) socket option.
	TCPKeepAlive time.Duration
}

// Server represents a server context which provides information about the
// running server and has control functions for managing state.
type Server struct {
	// The addrs parameter is an array of listening addresses that align
	// with the addr strings passed to the Serve function.
	Addrs []net.Addr
	// NumLoops is the number of loops that the server is using.
	NumLoops int
}

// Conn is an gnet connection.
type Conn interface {
	// Context returns a user-defined context.
	Context() interface{}
	// SetContext sets a user-defined context.
	SetContext(interface{})
	// AddrIndex is the index of server address that was passed to the Serve call.
	AddrIndex() int
	// LocalAddr is the connection's local socket address.
	LocalAddr() net.Addr
	// RemoteAddr is the connection's remote peer address.
	RemoteAddr() net.Addr
	// Wake triggers a React event for this connection.
	Wake()
	// Read reads all data from connection.
	Read() ([]byte, []byte)
	// Write writes data to connection.
	Write([]byte)
	// Advance advances the read pointer with the given number.
	Advance(int)
	// ResetBuffer reset the read pointer.
	ResetBuffer()
}

// Events represents the server events for the Serve call.
// Each event has an Action return value that is used manage the state
// of the connection and server.
type Events struct {
	// Multicore indicates whether the server will be effectively created with multi-cores, if so,
	// then you must take care with synchonizing memory between all event callbacks, otherwise,
	// it will run the server with single thread. The number of threads in the server will be automatically
	// assigned to the value of runtime.NumCPU().
	Multicore bool
	// OnInitComplete fires when the server can accept connections. The server
	// parameter has information and various utilities.
	OnInitComplete func(server Server) (action Action)
	// OnOpened fires when a new connection has opened.
	// The info parameter has information about the connection such as
	// it's local and remote address.
	// Use the out return value to write data to the connection.
	// The opts return value is used to set connection options.
	OnOpened func(c Conn) (opts Options, action Action)
	// OnClosed fires when a connection has closed.
	// The err parameter is the last known connection error.
	OnClosed func(c Conn, err error) (action Action)
	// PreWrite fires just before any data is written to any client socket.
	PreWrite func()
	// React fires when a connection sends the server data.
	// The in parameter is the incoming data.
	// Use the out return value to write data to the connection.
	React func(c Conn) (action Action)
	// Tick fires immediately after the server starts and will fire again
	// following the duration specified by the delay return value.
	Tick func() (delay time.Duration, action Action)
}

// Serve starts handling events for the specified addresses.
//
// Addresses should use a scheme prefix and be formatted
// like `tcp://192.168.0.10:9851` or `unix://socket`.
// Valid network schemes:
//  tcp   - bind to both IPv4 and IPv6
//  tcp4  - IPv4
//  tcp6  - IPv6
//  udp   - bind to both IPv4 and IPv6
//  udp4  - IPv4
//  udp6  - IPv6
//  unix  - Unix Domain Socket
//
// The "tcp" network scheme is assumed when one is not specified.
func Serve(events Events, addr ...string) error {
	var lns []*listener
	defer func() {
		for _, ln := range lns {
			ln.close()
		}
	}()

	var reusePort bool
	for _, addr := range addr {
		var ln listener
		ln.network, ln.addr, ln.opts = parseAddr(addr)
		if ln.network == "unix" {
			sniffError(os.RemoveAll(ln.addr))
		}
		reusePort = reusePort || ln.opts.reusePort
		var err error
		if ln.network == "udp" {
			if ln.opts.reusePort {
				ln.pconn, err = reuseportListenPacket(ln.network, ln.addr)
			} else {
				ln.pconn, err = net.ListenPacket(ln.network, ln.addr)
			}
		} else {
			if ln.opts.reusePort {
				ln.ln, err = reuseportListen(ln.network, ln.addr)
			} else {
				ln.ln, err = net.Listen(ln.network, ln.addr)
			}
		}
		if err != nil {
			return err
		}
		if ln.pconn != nil {
			ln.lnaddr = ln.pconn.LocalAddr()
		} else {
			ln.lnaddr = ln.ln.Addr()
		}
		if err := ln.system(); err != nil {
			return err
		}
		lns = append(lns, &ln)
	}
	return serve(events, lns, reusePort)
}

type listener struct {
	ln      net.Listener
	lnaddr  net.Addr
	pconn   net.PacketConn
	opts    addrOpts
	f       *os.File
	fd      int
	network string
	addr    string
}

type addrOpts struct {
	reusePort bool
}

func parseAddr(addr string) (network, address string, opts addrOpts) {
	network = "tcp"
	address = addr
	opts.reusePort = false
	if strings.Contains(address, "://") {
		network = strings.Split(address, "://")[0]
		address = strings.Split(address, "://")[1]
	}
	q := strings.Index(address, "?")
	if q != -1 {
		for _, part := range strings.Split(address[q+1:], "&") {
			kv := strings.Split(part, "=")
			if len(kv) == 2 {
				switch kv[0] {
				case "reuseport":
					if len(kv[1]) != 0 {
						switch kv[1][0] {
						default:
							opts.reusePort = kv[1][0] >= '1' && kv[1][0] <= '9'
						case 'T', 't', 'Y', 'y':
							opts.reusePort = true
						}
					}
				}
			}
		}
		address = address[:q]
	}
	return
}
