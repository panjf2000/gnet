// Copyright (c) 2025 The Gnet Authors. All rights reserved.
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

//go:build linux && !gnet_legacy_write

package io

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestWriteSocketNBSuppressesSIGPIPE(t *testing.T) {
	const childEnv = "GNET_TEST_SOCKET_NOSIGNAL_CHILD"
	if os.Getenv(childEnv) == "1" {
		sigpipe := make(chan os.Signal, 1)
		signal.Notify(sigpipe, syscall.SIGPIPE)
		defer signal.Stop(sigpipe)

		a, b := nonblockingPair(t)
		if err := unix.Close(b); err != nil {
			t.Fatal(err)
		}
		if n, err := WriteSocketNB(a, []byte("x")); n != -1 || err != unix.EPIPE {
			t.Fatalf("write after peer close: n=%d err=%v, want -1 / EPIPE", n, err)
		}
		select {
		case sig := <-sigpipe:
			t.Fatalf("unexpected signal after MSG_NOSIGNAL write: %v", sig)
		case <-time.After(25 * time.Millisecond):
		}
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestWriteSocketNBSuppressesSIGPIPE$")
	cmd.Env = append(os.Environ(), childEnv+"=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("SIGPIPE helper failed: %v\n%s", err, out)
	}
}
