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
		// Don't change the ordering of processing EPOLLOUT | EPOLLRDHUP / EPOLLIN unless you're 100%
		// sure what you're doing!
		// Re-ordering can easily introduce bugs and bad side-effects, as I found out painfully in the past.
		case !c.opened:
			return lp.loopOpen(c)
		case !c.outboundBuffer.IsEmpty():
			if ev&netpoll.OutEvents != 0 {
				return lp.loopOut(c)
			}
			return nil
		case ev&netpoll.InEvents != 0:
			return lp.loopIn(c)
		default:
			return nil
		}
	} else {
		return lp.loopAccept(fd)
	}
}
