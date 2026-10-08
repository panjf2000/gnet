// Copyright (c) 2026 Telego contributors.
// Licensed under the Apache License, Version 2.0. See LICENSE.

//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package gnet

import (
	"context"
	"net"
	"os"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/queue"
	"golang.org/x/sys/unix"
)

func (el *eventloop) ExecuteHighPriority(ctx context.Context, run Runnable) error {
	if el.engine.isShutdown() || el.engine.concurrency.ctx.Err() != nil {
		return errorx.ErrEngineInShutdown
	}
	if run == nil {
		return errorx.ErrNilRunnable
	}
	return el.poller.Trigger(queue.HighPriority, func(any) error { return run.Run(ctx) }, nil)
}

func (c *conn) WriteOwned(data []byte, release func(error)) (int, error) {
	if release == nil {
		release = func(error) {}
	}
	if !c.opened || c.loop.connections.getConn(c.fd) != c {
		release(net.ErrClosed)
		return 0, net.ErrClosed
	}
	if c.isDatagram {
		release(errorx.ErrUnsupportedOp)
		return 0, errorx.ErrUnsupportedOp
	}
	size := len(data)
	if size == 0 {
		release(nil)
		return 0, nil
	}
	empty := c.outboundBuffer.IsEmpty()
	sent := 0
	if empty {
		var err error
		sent, err = unix.Write(c.fd, data)
		if err != nil && err != unix.EAGAIN {
			cause := os.NewSyscallError("write", err)
			_ = c.loop.close(c, cause)
			release(cause)
			return 0, cause
		}
		if sent == size {
			release(nil)
			return size, nil
		}
		if sent < 0 {
			sent = 0
		}
	}
	// The callback retains the original allocation, even if only its suffix
	// remains. Ordinary writes append after this list node and preserve order.
	c.outboundBuffer.AppendOwned(data[sent:], release)
	if !empty {
		return size, nil
	}
	var err error
	if !c.loop.engine.opts.EdgeTriggeredIO {
		err = c.loop.poller.ModReadWrite(&c.pollAttachment, false)
	} else if sent > 0 {
		err = c.loop.poller.Trigger(queue.HighPriority, c.loop.write0, c)
	}
	if err != nil {
		_ = c.loop.close(c, err)
		return 0, err
	}
	return size, nil
}
