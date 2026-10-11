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

package netpoll

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/queue"
)

func assertPollerRetires(t *testing.T, start func(*Poller) error) {
	t.Helper()
	p, err := OpenPoller()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()

	var behind atomic.Bool
	if err = p.Trigger(queue.HighPriority, func(any) error { return errorx.ErrEngineShutdown }, nil); err != nil {
		t.Fatal(err)
	}
	if err = p.Trigger(queue.LowPriority, func(any) error {
		behind.Store(true)
		return nil
	}, nil); err != nil {
		t.Fatal(err)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- start(p) }()
	select {
	case err = <-errCh:
		if !errors.Is(err, errorx.ErrEngineShutdown) {
			t.Fatalf("Polling returned %v, want ErrEngineShutdown", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Polling still blocked 3s after shutdown task")
	}
	if behind.Load() {
		t.Fatal("task queued behind shutdown was executed")
	}
	if !p.urgentAsyncTaskQueue.IsEmpty() || !p.asyncTaskQueue.IsEmpty() {
		t.Fatal("retired poller left tasks queued")
	}
	select {
	case <-p.Done():
	default:
		t.Fatal("Done was not closed when Polling returned")
	}

	var after atomic.Bool
	err = p.Trigger(queue.HighPriority, func(any) error {
		after.Store(true)
		return nil
	}, nil)
	if !errors.Is(err, errorx.ErrEngineShutdown) {
		t.Fatalf("Trigger after retire returned %v, want ErrEngineShutdown", err)
	}
	if after.Load() {
		t.Fatal("Trigger after retire executed the task")
	}
}
