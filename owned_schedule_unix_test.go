// Copyright (c) 2026 Telego contributors.
// Licensed under the Apache License, Version 2.0. See LICENSE.

//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package gnet

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
)

func TestOwnedWriteScheduledAfterAsyncWrite(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	client, err := NewClient(&BuiltinEventEngine{}, WithNumEventLoop(1))
	if err != nil {
		t.Fatal(err)
	}
	if err = client.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Stop() }()
	conn, err := client.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	peer, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = peer.Close() }()
	owner := conn.EventLoop()
	ctx := context.Background()
	if err = ExecuteHighPriority(owner, ctx, nil); !errors.Is(err, errorx.ErrNilRunnable) {
		t.Fatalf("nil runnable: %v", err)
	}
	wrapped := struct{ EventLoop }{owner}
	if err = ExecuteHighPriority(wrapped, ctx, RunnableFunc(func(context.Context) error { return nil })); !errors.Is(err, errorx.ErrUnsupportedOp) {
		t.Fatalf("unsupported wrapper: %v", err)
	}
	if err = conn.AsyncWrite([]byte("copied/"), nil); err != nil {
		t.Fatal(err)
	}
	released := make(chan error, 1)
	if err = ExecuteHighPriority(owner, ctx, RunnableFunc(func(context.Context) error {
		_, writeErr := conn.(OwnedWriter).WriteOwned([]byte("borrowed"), func(err error) { released <- err })
		return writeErr
	})); err != nil {
		t.Fatal(err)
	}
	if err = peer.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, len("copied/borrowed"))
	if _, err = io.ReadFull(peer, data); err != nil {
		t.Fatal(err)
	}
	if string(data) != "copied/borrowed" {
		t.Fatalf("write order: %q", data)
	}
	select {
	case err = <-released:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("owned write did not release its allocation")
	}
}
