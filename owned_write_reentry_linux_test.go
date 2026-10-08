// Copyright (c) 2026 Telego contributors.
// Licensed under the Apache License, Version 2.0. See LICENSE.

package gnet

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/panjf2000/gnet/v2/pkg/netpoll"
	"golang.org/x/sys/unix"
)

// The socket pair exercises real writev and connection callbacks without a
// listener, poller worker, or competing TCP port allocation.
func ownedReentryConn(t *testing.T) (*conn, int) {
	t.Helper()
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_NONBLOCK|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	poller, err := netpoll.OpenPoller()
	if err != nil {
		_ = unix.Close(fds[0])
		_ = unix.Close(fds[1])
		t.Fatal(err)
	}
	el := &eventloop{
		engine:       &engine{opts: &Options{WriteBufferCap: 1024}},
		poller:       poller,
		eventHandler: &BuiltinEventEngine{},
	}
	el.connections.init()
	c := newStreamConn("unix", fds[0], el, nil, nil, nil)
	c.opened = true
	el.connections.addConn(c, 0)
	t.Cleanup(func() {
		if c.opened {
			_ = el.close(c, nil)
		}
		_ = unix.Close(fds[1])
		_ = poller.Close()
	})
	if err := poller.AddRead(&c.pollAttachment, false); err != nil {
		t.Fatal(err)
	}
	return c, fds[1]
}

func TestOwnedWriteBoundedDescriptorsPreserveReentry(t *testing.T) {
	for _, operation := range []string{"flush", "close"} {
		t.Run(operation, func(t *testing.T) {
			c, peer := ownedReentryConn(t)
			const count = 3*iovMax + 17
			released := make([]int, count)
			expected := make([]byte, 1+4*count)
			expected[0] = 'P'
			_, _ = c.outboundBuffer.Write(expected[:1])
			for index := 0; index < count; index++ {
				index := index
				data := make([]byte, 4)
				binary.LittleEndian.PutUint32(data, uint32(index))
				copy(expected[1+4*index:], data)
				if _, err := c.WriteOwned(data, func(err error) {
					released[index]++
					if err != nil {
						t.Errorf("release %d: %v", index, err)
					}
					for _, retained := range c.loop.writeBuffers {
						if retained != nil {
							t.Fatal("owner scratch retained output during release callback")
						}
					}
					if index == 0 {
						if operation == "close" {
							err = c.loop.Close(c)
						} else {
							err = c.Flush()
						}
						if err != nil {
							t.Errorf("reentrant %s: %v", operation, err)
						}
					}
				}); err != nil {
					t.Fatal(err)
				}
			}
			for c.opened && c.OutboundBuffered() != 0 {
				if err := c.Flush(); err != nil {
					t.Fatal(err)
				}
			}
			got := make([]byte, len(expected))
			offset := 0
			for offset < len(got) {
				n, err := unix.Read(peer, got[offset:])
				if err != nil || n <= 0 {
					t.Fatalf("socket drain at %d: %d, %v", offset, n, err)
				}
				offset += n
			}
			if !bytes.Equal(got, expected) {
				t.Fatal("bounded writev changed or repeated response bytes")
			}
			for index, calls := range released {
				if calls != 1 {
					t.Fatalf("release %d ran %d times", index, calls)
				}
			}
			for _, retained := range c.loop.writeBuffers {
				if retained != nil {
					t.Fatal("owner retained descriptor after drain")
				}
			}
		})
	}
}

func TestOwnedWriteReleaseReentryDoesNotRepeatWrittenBatch(t *testing.T) {
	for _, operation := range []string{"flush", "close"} {
		t.Run(operation, func(t *testing.T) {
			c, peer := ownedReentryConn(t)
			// A preceding ordinary write forces both borrowed writes to queue
			// together. One writev sends XAB before either release callback.
			if _, err := c.outboundBuffer.Write([]byte("X")); err != nil {
				t.Fatal(err)
			}
			var callbacks int
			if n, err := c.WriteOwned([]byte("A"), func(err error) {
				callbacks++
				if err != nil {
					t.Errorf("release A: %v", err)
				}
				if buffered := c.OutboundBuffered(); buffered != 0 {
					t.Errorf("release saw %d bytes already sent by outer writev", buffered)
				}
				if operation == "close" {
					err = c.loop.Close(c)
				} else {
					err = c.Flush()
				}
				if err != nil {
					t.Errorf("reentrant %s: %v", operation, err)
				}
			}); n != 1 || err != nil {
				t.Fatalf("queue A: n=%d err=%v", n, err)
			}
			if n, err := c.WriteOwned([]byte("B"), func(err error) {
				callbacks++
				if err != nil {
					t.Errorf("release B: %v", err)
				}
			}); n != 1 || err != nil {
				t.Fatalf("queue B: n=%d err=%v", n, err)
			}
			if callbacks != 0 {
				t.Fatalf("released %d callbacks before socket drain", callbacks)
			}
			if err := c.Flush(); err != nil {
				t.Fatalf("outer flush after reentrant %s: %v", operation, err)
			}
			if callbacks != 2 {
				t.Fatalf("callbacks=%d, want 2", callbacks)
			}
			var received [16]byte
			n, err := unix.Read(peer, received[:])
			if err != nil {
				t.Fatal(err)
			}
			if got := string(received[:n]); got != "XAB" {
				t.Fatalf("socket received %q, want XAB exactly once", got)
			}
		})
	}
}
