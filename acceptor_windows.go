// Copyright (c) 2023 The Gnet Authors. All rights reserved.
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
	"errors"
	"net"
	"runtime"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
)

func (eng *engine) listenStream(ln net.Listener) (err error) {
	if eng.opts.LockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	defer func() { eng.shutdown(err) }()

	for {
		// Accept TCP socket.
		tc, e := ln.Accept()
		if e != nil {
			err = e
			if !eng.beingShutdown.Load() {
				eng.opts.Logger.Errorf("Accept() fails due to error: %v", err)
			} else if errors.Is(err, net.ErrClosed) {
				err = errors.Join(err, errorx.ErrEngineShutdown)
			}
			return
		}
		el := eng.eventLoops.next(tc.RemoteAddr())
		c := newTCPConn(tc, el)
		el.ch <- &openConn{c: c}
		go func(c *conn, tc net.Conn, el *eventloop) {
			var buffer [0x10000]byte
			for {
				n, err := tc.Read(buffer[:])
				if err != nil {
					el.ch <- &netErr{c, err}
					return
				}
				el.ch <- packTCPConn(c, buffer[:n])
			}
		}(c, tc, el)
	}
}

func (eng *engine) ListenUDP(pc net.PacketConn) (err error) {
	if eng.opts.LockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	defer func() { eng.shutdown(err) }()

	var buffer [0x10000]byte
	for {
		// Read data from UDP socket.
		n, addr, e := pc.ReadFrom(buffer[:])
		if e != nil {
			err = e
			if !eng.beingShutdown.Load() {
				eng.opts.Logger.Errorf("failed to receive data from UDP fd due to error:%v", err)
			} else if errors.Is(err, net.ErrClosed) {
				err = errors.Join(err, errorx.ErrEngineShutdown)
			}
			return
		}
		el := eng.eventLoops.next(addr)
		c := newUDPConn(el, pc, pc.LocalAddr(), addr)
		el.ch <- packUDPConn(c, buffer[:n])
	}
}
