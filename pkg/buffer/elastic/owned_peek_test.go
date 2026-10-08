// Copyright (c) 2026 Telego contributors.
// Licensed under the Apache License, Version 2.0. See LICENSE.

package elastic

import (
	"encoding/binary"
	"testing"
)

func TestOwnedBufferPeekIntoBoundsTinyBacklog(t *testing.T) {
	buffer, err := New(32)
	if err != nil {
		t.Fatal(err)
	}
	defer buffer.Release()
	const count = 100000
	released := make([]int, count)
	_, _ = buffer.Write([]byte("P"))
	for index := 0; index < count; index++ {
		index := index
		data := make([]byte, 4)
		binary.LittleEndian.PutUint32(data, uint32(index))
		buffer.AppendOwned(data, func(err error) {
			released[index]++
			if err != nil {
				t.Errorf("release %d: %v", index, err)
			}
		})
	}
	if n := buffer.PeekInto(nil); n != 0 {
		t.Fatalf("empty destination returned %d descriptors", n)
	}
	// The existing public API still returns every descriptor when requested.
	all, err := buffer.Peek(-1)
	if err != nil || len(all) != count+1 {
		t.Fatalf("public Peek changed: %d descriptors, %v", len(all), err)
	}
	var scratch [1024][]byte
	if allocations := testing.AllocsPerRun(100, func() {
		if n := buffer.PeekInto(scratch[:]); n != len(scratch) {
			t.Fatalf("bounded peek returned %d descriptors", n)
		}
	}); allocations != 0 {
		t.Fatalf("bounded peek allocated %.1f objects", allocations)
	}
	seen := -1
	for !buffer.IsEmpty() {
		n := buffer.PeekInto(scratch[:])
		if n == 0 || n > len(scratch) {
			t.Fatalf("invalid descriptor count %d", n)
		}
		bytes := 0
		for i, data := range scratch[:n] {
			if seen < 0 {
				if string(data) != "P" {
					t.Fatal("ring prefix moved after owned output")
				}
			} else if len(data) != 4 || binary.LittleEndian.Uint32(data) != uint32(seen) {
				t.Fatalf("owned output changed at %d", seen)
			}
			seen++
			bytes += len(data)
			scratch[i] = nil
		}
		if discarded, err := buffer.Discard(bytes); err != nil || discarded != bytes {
			t.Fatalf("discard = %d, %v; want %d", discarded, err, bytes)
		}
	}
	if seen != count {
		t.Fatalf("drained %d owned entries, want %d", seen, count)
	}
	for index, calls := range released {
		if calls != 1 {
			t.Fatalf("release %d ran %d times", index, calls)
		}
	}
	for _, data := range scratch {
		if data != nil {
			t.Fatal("scratch retained released output")
		}
	}
}
