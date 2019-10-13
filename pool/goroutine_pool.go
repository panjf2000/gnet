// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package pool

import (
	"time"

	"github.com/panjf2000/ants/v2"
)

const (
	// DefaultAntsPoolSize sets up the capacity of worker pool.
	DefaultAntsPoolSize = 256 * 1024

	// ExpiryDuration is the interval time to clean up those expired workers.
	ExpiryDuration = 10 * time.Second

	// Nonblocking decides what to do when submitting a new job to a full worker pool: waiting for a available worker
	// or returning nil directly.
	Nonblocking = true
)

func init() {
	// It releases the default pool from ants.
	ants.Release()
}

// WorkerPool is the alias of ants.Pool.
type WorkerPool = ants.Pool

// NewWorkerPool instantiates a non-blocking *WorkerPool with the capacity of DefaultAntsPoolSize.
func NewWorkerPool() *WorkerPool {
	options := ants.Options{ExpiryDuration: ExpiryDuration, Nonblocking: Nonblocking}
	defaultAntsPool, _ := ants.NewPool(DefaultAntsPoolSize, ants.WithOptions(options))
	return defaultAntsPool
}
