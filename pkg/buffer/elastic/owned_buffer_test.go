// Copyright (c) 2026 Telego contributors.
// Licensed under the Apache License, Version 2.0. See LICENSE.

package elastic

import (
	"bytes"
	"io"
	"testing"
)

func TestOwnedBufferMixedOrderAndPartialDisposal(t *testing.T) {
	buffer, err := New(32)
	if err != nil {
		t.Fatal(err)
	}
	defer buffer.Release()
	_, _ = buffer.Write([]byte("prefix"))
	owned := bytes.Repeat([]byte{'A'}, 65537)
	released := 0
	buffer.AppendOwned(owned, func(err error) {
		released++
		if err != nil {
			t.Errorf("release: %v", err)
		}
		if buffer.Buffered() != len("tail") {
			t.Errorf("release saw uncommitted byte count %d", buffer.Buffered())
		}
	})
	_, _ = buffer.Write([]byte("tail"))
	peek, err := buffer.Peek(-1)
	if err != nil {
		t.Fatal(err)
	}
	want := append([]byte("prefix"), owned...)
	want = append(want, []byte("tail")...)
	if !bytes.Equal(bytes.Join(peek, nil), want) {
		t.Fatal("mixed normal/owned writes changed order")
	}
	if len(peek) < 2 || &peek[1][0] != &owned[0] {
		t.Fatal("owned slice was copied into another allocation")
	}
	_, _ = buffer.Discard(len("prefix") + len(owned) - 1)
	if released != 0 || buffer.Buffered() != 1+len("tail") {
		t.Fatal("partial discard disposed owned allocation")
	}
	_, _ = buffer.Discard(1)
	if released != 1 {
		t.Fatal("final discard did not dispose owned allocation")
	}
	buffer.Release()
	if released != 1 {
		t.Fatal("release disposed allocation twice")
	}
}

func TestOwnedBufferReleaseUnlinksBeforeReentry(t *testing.T) {
	buffer, _ := New(32)
	defer buffer.Release()
	called := 0
	buffer.AppendOwned(make([]byte, 17), func(err error) {
		called++
		if err != io.ErrClosedPipe || buffer.Buffered() != 0 {
			t.Error("release callback ran before complete unlink")
		}
		_, _ = buffer.Write([]byte("new"))
	})
	buffer.AppendOwned(make([]byte, 33), func(err error) {
		called++
		if err != io.ErrClosedPipe {
			t.Error("abandoned buffer reported success")
		}
	})
	buffer.Release()
	peek, _ := buffer.Peek(-1)
	if called != 2 || string(bytes.Join(peek, nil)) != "new" {
		t.Fatal("reentrant write lost or repeated disposal")
	}
}

type ownedShortWriter struct{}

func (ownedShortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }

func TestOwnedBufferReadAndWriteToKeepPartialAllocation(t *testing.T) {
	for _, operation := range []string{"read", "write_to"} {
		t.Run(operation, func(t *testing.T) {
			buffer, _ := New(32)
			defer buffer.Release()
			called := 0
			buffer.AppendOwned(make([]byte, 33), func(error) { called++ })
			if operation == "read" {
				if n, err := buffer.Read(make([]byte, 32)); n != 32 || err != nil {
					t.Fatalf("read=%d,%v", n, err)
				}
			} else {
				if n, err := buffer.WriteTo(ownedShortWriter{}); n != 32 || err != io.ErrShortWrite {
					t.Fatalf("write=%d,%v", n, err)
				}
			}
			if called != 0 || buffer.Buffered() != 1 {
				t.Fatal("partial operation disposed whole allocation")
			}
			if n, err := buffer.Read(make([]byte, 1)); n != 1 || err != nil {
				t.Fatalf("last read=%d,%v", n, err)
			}
			if called != 1 {
				t.Fatal("last byte did not release allocation")
			}
		})
	}
}
