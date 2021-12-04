// Copyright (c) 2019 Andy Pan
// Copyright (c) 2018 Joshua J Baker
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
	"runtime"
	"time"
)

func (svr *server) listenerRun(lockOSThread bool) {
	if lockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	var err error
	defer func() { svr.signalShutdownWithErr(err) }()
	var buffer [0x10000]byte
	for {
		if svr.ln.packetConn != nil {
			// Read data from UDP socket.
			n, addr, e := svr.ln.packetConn.ReadFrom(buffer[:])
			if e != nil {
				err = e
				svr.opts.Logger.Errorf("failed to receive data from UDP fd due to error:%v", err)
				return
			}

			el := svr.lb.next(addr)
			c := newUDPConn(el, svr.ln.addr, addr)
			el.ch <- packUDPConn(c, buffer[:n])
		} else {
			// Accept TCP socket.
			conn, e := svr.ln.ln.Accept()
			if e != nil {
				err = e
				svr.opts.Logger.Errorf("Accept() fails due to error: %v", err)
				return
			}
			el := svr.lb.next(conn.RemoteAddr())
			c := newTCPConn(conn, el)
			el.ch <- c
			go func() {
				var buffer [0x10000]byte
				for {
					n, err := conn.Read(buffer[:])
					if err != nil {
						_ = conn.SetReadDeadline(time.Time{})
						el.ch <- &stderr{c, err}
						return
					}
					el.ch <- packTCPConn(c, buffer[:n])
				}
			}()
		}
	}
}
