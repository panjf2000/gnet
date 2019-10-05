// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build darwin netbsd freebsd openbsd dragonfly

package gnet

import (
	"github.com/panjf2000/gnet/internal"
	"github.com/panjf2000/gnet/netpoll"
)

func (lp *loop) handleEvent(fd int, filter int16, job internal.Job) error {
	if fd == 0 {
		return job()
	}
	if c, ok := lp.connections[fd]; ok {
		switch {
		case !c.opened:
			return lp.loopOpen(c)
		case !c.outboundBuffer.IsEmpty():
			if filter == netpoll.EVFilterWrite {
				return lp.loopOut(c)
			}
		case filter == netpoll.EVFilterRead:
			return lp.loopIn(c)
		case filter == netpoll.EVFilterSock:
			return lp.loopCloseConn(c, nil)
		}
	} else {
		return lp.loopAccept(fd)
	}
	return nil
}
