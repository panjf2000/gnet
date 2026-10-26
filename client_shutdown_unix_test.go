// Copyright (c) 2026 The Gnet Authors. All rights reserved.
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

//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package gnet

import (
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
)

func TestEnrollAfterStopReturns(t *testing.T) {
	cli, err := NewClient(&BuiltinEventEngine{})
	if err != nil {
		t.Fatal(err)
	}
	if err = cli.Start(); err != nil {
		t.Fatal(err)
	}
	if err = cli.Stop(); err != nil {
		t.Fatal(err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	raw, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	before := openFDCount(t)
	done := make(chan error, 1)
	go func() {
		_, enrollErr := cli.Enroll(raw)
		done <- enrollErr
	}()
	select {
	case err = <-done:
		if err == nil {
			t.Fatal("Enroll after Stop succeeded")
		}
		if !errors.Is(err, errorx.ErrEngineInShutdown) && !errors.Is(err, errorx.ErrEngineShutdown) {
			t.Fatalf("Enroll after Stop: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Enroll still blocked 3s after Stop")
	}
	// Enroll always closes the caller's socket on return. The duplicated
	// descriptor must not remain, so the count drops by exactly one.
	if after := openFDCount(t); before >= 0 && after != before-1 {
		t.Fatalf("open descriptors %d -> %d across Enroll after Stop, want %d", before, after, before-1)
	}
}

// TestCloseUnopenedKeepsFD checks that eventloop.close never closes the
// descriptor of an unopened conn. Such a conn is either already closed, in
// which case the kernel may have reused the number, or a UDP listener's
// per-packet conn sharing the listener fd.
func TestCloseUnopenedKeepsFD(t *testing.T) {
	fd, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}
	el := &eventloop{}
	el.connections.init()
	c := &conn{fd: fd}
	for i := 0; i < 2; i++ {
		if err = el.close(c, nil); err != nil {
			t.Fatal(err)
		}
	}
	if err = unix.Close(fd); err != nil {
		t.Fatalf("close of an unopened conn released fd=%d: %v", fd, err)
	}
}

func openFDCount(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		if os.IsNotExist(err) {
			return -1
		}
		t.Fatal(err)
	}
	return len(entries)
}
