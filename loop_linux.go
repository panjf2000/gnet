// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build linux

package gnet

import (
	"github.com/panjf2000/gnet/internal"
	"github.com/panjf2000/gnet/netpoll"
)

func (lp *loop) handleEvent(fd int, ev uint32, job internal.Job) error {
	if fd == 0 {
		return job()
	}
	if c, ok := lp.connections[fd]; ok {
		switch {
		case !c.opened:
			return lp.loopOpen(c)
		case !c.outboundBuffer.IsEmpty():
			if ev&netpoll.OutEvents != 0 {
				return lp.loopOut(c)
			}
		case ev&netpoll.InEvents != 0:
			return lp.loopIn(c)
		}
	} else {
		return lp.loopAccept(fd)
	}
	return nil
}
