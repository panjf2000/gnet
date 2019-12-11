// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package ringbuffer

import (
	"sync"

	"github.com/panjf2000/gnet/ringbuffer"
)

const (
	// DefaultRingBufferSize represents the initial size of connection ring-buffer.
	DefaultRingBufferSize = 1 << 12
)

// pool for caching ring-buffers.
var pool sync.Pool

func init() {
	pool.New = func() interface{} {
		return ringbuffer.New(DefaultRingBufferSize)
	}
}

// Get gets ring-buffer from pool.
func Get() *ringbuffer.RingBuffer {
	return pool.Get().(*ringbuffer.RingBuffer)
}

// Put puts ring-buffer back into pool.
func Put(rb *ringbuffer.RingBuffer) {
	rb.Reset()
	pool.Put(rb)
}
