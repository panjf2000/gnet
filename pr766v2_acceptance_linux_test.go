//go:build linux

package gnet

import (
	"errors"
	"net"
	"testing"
	"time"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
)

func pr766v2TCPPair(t *testing.T) *net.TCPConn {
	t.Helper()
	ln, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := ln.Close(); err != nil {
			t.Error(err)
		}
	})
	raw, err := net.DialTCP("tcp", nil, ln.Addr().(*net.TCPAddr))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := raw.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Error(err)
		}
	})
	peer, err := ln.AcceptTCP()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := peer.Close(); err != nil {
			t.Error(err)
		}
	})
	return raw
}

func TestPR766V2OriginalSocketOracle(t *testing.T) {
	raw := pr766v2TCPPair(t)
	rc, err := raw.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	if err := rc.Control(func(uintptr) {}); err != nil {
		t.Fatalf("live socket Control = %v, want nil", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := rc.Control(func(uintptr) {}); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("closed socket Control = %v, want net.ErrClosed", err)
	}
}

func TestPR766V2EnrollWorkerRejectClosesOriginal(t *testing.T) {
	raw := pr766v2TCPPair(t)
	rc, err := raw.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	if err := rc.Control(func(uintptr) {}); err != nil {
		t.Fatal(err)
	}
	el := &eventloop{engine: &engine{}}
	// Model the worker boundary after public admission and before execution.
	// No pool-wide hook or scheduler assumption is needed.
	el.engine.stopping.Store(true)
	results, err := el.enroll(raw, raw.RemoteAddr(), nil)
	if err != nil {
		t.Fatalf("worker submission = %v, want nil", err)
	}
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	select {
	case result, open := <-results:
		if !open || result.Conn != nil || !errors.Is(result.Err, errorx.ErrEngineInShutdown) {
			t.Fatalf("result = %+v, channel open = %v, want nil connection and ErrEngineInShutdown", result, open)
		}
	case <-timer.C:
		t.Fatal("worker did not publish its rejection")
	}
	select {
	case _, open := <-results:
		if open {
			t.Fatal("worker published more than one result")
		}
	case <-timer.C:
		t.Fatal("worker did not finish after rejection")
	}
	if err := rc.Control(func(uintptr) {}); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("original socket after worker rejection: Control = %v, want net.ErrClosed", err)
	}
}

func TestPR766V2AwaitRegisterCompletionWins(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		opened bool
		done   bool
		want   bool
	}{
		{"both_ready", true, true, true},
		{"only_opened", true, false, true},
		{"only_retired", false, true, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			opened, done := make(chan struct{}), make(chan struct{})
			if scenario.opened {
				close(opened)
			}
			if scenario.done {
				close(done)
			}
			// Repetition exercises select's two-ready arbitration; the contract
			// must hold for every choice, not merely the first observed one.
			for i := 0; i < 1024; i++ {
				if got := awaitPollerRegister(opened, done); got != scenario.want {
					t.Fatalf("iteration %d: completion = %v, want %v", i, got, scenario.want)
				}
			}
		})
	}
}
