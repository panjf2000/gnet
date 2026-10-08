// Copyright (c) 2026 Telego contributors.
// Licensed under the Apache License, Version 2.0. See LICENSE.

//go:build linux

package netpoll

import (
	"testing"

	"github.com/panjf2000/gnet/v2/pkg/queue"
)

// BenchmarkTriggerCoalesced isolates submission while a wakeup is pending.
// The consumer drains each task so the benchmark retains bounded memory.
// It excludes socket I/O and is not an end-to-end throughput measurement.
func BenchmarkTriggerCoalesced(b *testing.B) {
	poller, err := OpenPoller()
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = poller.Close() }()
	callback := func(any) error { return nil }
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := poller.Trigger(queue.HighPriority, callback, nil); err != nil {
			b.Fatal(err)
		}
		queue.PutTask(poller.urgentAsyncTaskQueue.Dequeue())
	}
	b.StopTimer()
}
