// Copyright (c) 2026 Telego contributors.
// Licensed under the Apache License, Version 2.0. See LICENSE.

package gnet

import (
	"bytes"
	"testing"

	"golang.org/x/sys/unix"
)

func TestOwnedWriteRealPartialSocketDrainRetainsAllocation(t *testing.T) {
	for _, finish := range []string{"drain", "close"} {
		for _, edge := range []bool{false, true} {
			t.Run(finish+map[bool]string{false: "_lt", true: "_et"}[edge], func(t *testing.T) {
				c, peer := ownedReentryConn(t)
				c.loop.engine.opts.EdgeTriggeredIO = edge
				if err := unix.SetsockoptInt(c.fd, unix.SOL_SOCKET, unix.SO_SNDBUF, 4096); err != nil {
					t.Fatal(err)
				}
				payload := bytes.Repeat([]byte{'Q'}, 65537)
				charged, callbacks := cap(payload), 0
				var releaseErr error
				if n, err := c.WriteOwned(payload, func(err error) { charged = 0; callbacks++; releaseErr = err }); n != len(payload) || err != nil {
					t.Fatalf("write=%d,%v", n, err)
				}
				before := c.OutboundBuffered()
				if before <= 0 || before >= len(payload) || charged != cap(payload) || callbacks != 0 {
					t.Fatalf("socket did not partially retain write: buffered=%d charged=%d callbacks=%d", before, charged, callbacks)
				}
				var received bytes.Buffer
				chunk := make([]byte, 8192)
				read, err := unix.Read(peer, chunk)
				if err != nil || read <= 0 {
					t.Fatalf("partial peer read=%d,%v", read, err)
				}
				received.Write(chunk[:read])
				if err := c.Flush(); err != nil {
					t.Fatal(err)
				}
				if buffered := c.OutboundBuffered(); buffered <= 0 || buffered >= before || charged != cap(payload) || callbacks != 0 {
					t.Fatalf("partial flush lost charge: before=%d after=%d charged=%d", before, buffered, charged)
				}
				if finish == "close" {
					if err := c.loop.Close(c); err != nil {
						t.Fatal(err)
					}
					if callbacks != 1 || charged != 0 || releaseErr == nil {
						t.Fatal("abandoned socket output was not disposed once")
					}
					return
				}
				for attempts := 0; attempts < 100 && received.Len() < len(payload); attempts++ {
					read, err = unix.Read(peer, chunk)
					if err != nil && err != unix.EAGAIN {
						t.Fatal(err)
					}
					if read > 0 {
						received.Write(chunk[:read])
					}
					if err := c.Flush(); err != nil {
						t.Fatal(err)
					}
				}
				if !bytes.Equal(received.Bytes(), payload) || callbacks != 1 || charged != 0 || releaseErr != nil {
					t.Fatalf("full drain: received=%d callback=%d charge=%d error=%v", received.Len(), callbacks, charged, releaseErr)
				}
			})
		}
	}
}
