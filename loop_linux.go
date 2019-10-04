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
		if !c.opened {
			return lp.loopOpen(c)
		}
		if ev&netpoll.InEvents != 0 {
			if err := lp.loopIn(c); err != nil {
				return err
			}
		}
		if ev&netpoll.OutEvents != 0 && !c.outboundBuffer.IsEmpty() {
			if err := lp.loopOut(c); err != nil {
				return err
			}
		}
	} else {
		return lp.loopAccept(fd)
	}
	return nil
}
