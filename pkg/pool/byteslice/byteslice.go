// Copyright (c) 2021 Andy Pan
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

package byteslice

import (
	"math"
	"math/bits"
	"reflect"
	"runtime"
	"sync"
	"unsafe"
)

var builtinPool Pool

// Pool consists of 32 sync.Pool, representing byte slices of length from 0 to 32 in powers of 2.
type Pool struct {
	pools [32]sync.Pool
}

// Get returns a byte slice with given length from the built-in pool.
func Get(size int) []byte {
	return builtinPool.Get(size)
}

// Put returns the byte slice to the built-in pool.
func Put(buf []byte) {
	builtinPool.Put(buf)
}

// Get retrieves a byte slice of the length requested by the caller from pool or allocates a new one.
func (p *Pool) Get(size int) (buf []byte) {
	if size <= 0 {
		return nil
	}
	if size > math.MaxInt32 {
		return make([]byte, size)
	}
	idx := index(uint32(size))
	ptr, _ := p.pools[idx].Get().(unsafe.Pointer)
	if ptr == nil {
		return make([]byte, 1<<idx)[:size]
	}
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	sh.Data = uintptr(ptr)
	sh.Len = size
	sh.Cap = 1 << idx
	runtime.KeepAlive(ptr)
	return
}

// Put returns the byte slice to the pool.
func (p *Pool) Put(buf []byte) {
	size := cap(buf)
	if size == 0 || size > math.MaxInt32 {
		return
	}
	idx := index(uint32(size))
	if size != 1<<idx { // this byte slice is not from Pool.Get(), put it into the previous interval of idx
		idx--
	}
	// array pointer
	p.pools[idx].Put(unsafe.Pointer(&buf[:1][0]))
}

func index(n uint32) uint32 {
	return uint32(bits.Len32(n - 1))
}
