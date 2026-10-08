// Copyright (c) 2026 Telego contributors.
// Licensed under the Apache License, Version 2.0. See LICENSE.

//go:build linux

package gnet_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/panjf2000/gnet/v2"
	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
)

func regressionTCPPair(t *testing.T) (net.Conn, net.Conn) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	client, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	peer, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = peer.Close() })
	return client, peer
}

func regressionSocketName(t *testing.T, c net.Conn) string {
	t.Helper()
	raw, err := c.(syscall.Conn).SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	var name string
	if err = raw.Control(func(fd uintptr) { name, err = os.Readlink(fmt.Sprintf("/proc/self/fd/%d", fd)) }); err != nil {
		t.Fatal(err)
	}
	if name == "" {
		t.Fatal("socket descriptor has no procfs identity")
	}
	return name
}

func regressionSocketCopies(t *testing.T, name string) int {
	t.Helper()
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Fatal(err)
	}
	copies := 0
	for _, entry := range entries {
		value, _ := os.Readlink("/proc/self/fd/" + entry.Name())
		if value == name {
			copies++
		}
	}
	return copies
}

// This test intentionally uses only upstream v2.10.0 public APIs. Observing
// two descriptors for the source socket proves duplication happened before
// the owner exits; successful queue admission alone does not prove that.
func TestAcceptedEnrollmentOwnerExitDisposesSocket(t *testing.T) {
	for _, synchronous := range []bool{false, true} {
		t.Run(fmt.Sprintf("client_enroll_%t", synchronous), func(t *testing.T) {
			client, err := gnet.NewClient(&gnet.BuiltinEventEngine{}, gnet.WithNumEventLoop(1))
			if err != nil {
				t.Fatal(err)
			}
			if err = client.Start(); err != nil {
				t.Fatal(err)
			}
			var stopOnce sync.Once
			stopped := make(chan error, 1)
			stop := func() { stopOnce.Do(func() { go func() { stopped <- client.Stop() }() }) }
			t.Cleanup(stop)
			bootstrap, _ := regressionTCPPair(t)
			connection, err := client.Enroll(bootstrap)
			if err != nil {
				t.Fatal(err)
			}
			owner := connection.EventLoop()
			entered, release := make(chan struct{}), make(chan struct{})
			var releaseOnce sync.Once
			unblock := func() { releaseOnce.Do(func() { close(release) }) }
			t.Cleanup(unblock)
			if err = owner.Execute(context.Background(), gnet.RunnableFunc(func(context.Context) error {
				close(entered)
				<-release
				return errorx.ErrEngineShutdown
			})); err != nil {
				t.Fatal(err)
			}
			select {
			case <-entered:
			case <-time.After(2 * time.Second):
				t.Fatal("owner did not enter pause")
			}
			source, peer := regressionTCPPair(t)
			identity := regressionSocketName(t, source)
			var result <-chan gnet.RegisteredResult
			if synchronous {
				channel := make(chan gnet.RegisteredResult, 1)
				result = channel
				go func() {
					c, enrollErr := client.EnrollContext(source, nil)
					channel <- gnet.RegisteredResult{Conn: c, Err: enrollErr}
					close(channel)
				}()
			} else {
				result, err = owner.Enroll(context.Background(), source)
				if err != nil {
					t.Fatal(err)
				}
			}
			deadline := time.Now().Add(2 * time.Second)
			for regressionSocketCopies(t, identity) != 2 {
				if time.Now().After(deadline) {
					t.Fatal("accepted enrollment never duplicated its socket")
				}
				runtime.Gosched()
			}
			unblock()
			stop()
			select {
			case stopErr := <-stopped:
				if stopErr != nil {
					t.Fatal(stopErr)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("client shutdown stranded an accepted enrollment")
			}
			select {
			case got, ok := <-result:
				if !ok || got.Err == nil || got.Conn != nil {
					t.Fatalf("enrollment result = %#v, open=%v", got, ok)
				}
			case <-time.After(time.Second):
				t.Fatal("accepted enrollment has no terminal result after owner exit")
			}
			if copies := regressionSocketCopies(t, identity); copies != 0 {
				t.Fatalf("shutdown retained %d socket descriptors", copies)
			}
			_ = peer.SetReadDeadline(time.Now().Add(time.Second))
			var b [1]byte
			if n, readErr := peer.Read(b[:]); n != 0 || readErr != io.EOF {
				t.Fatalf("peer read = %d, %v; want EOF", n, readErr)
			}
		})
	}
}
