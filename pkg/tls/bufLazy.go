package tls

import (
	"sync"
)

const defaultSize = 4096

var bytePool = sync.Pool{
	New: func() any {
		buf := make([]byte, defaultSize)
		return &buf
	},
}

type LazyBuffer struct {
	buf []byte
	ref *[]byte
}

func (lb *LazyBuffer) Bytes() []byte {
	return lb.buf
}

func (lb *LazyBuffer) Len() int {
	return len(lb.buf)
}

func (lb *LazyBuffer) Next(n int) []byte {
	m := lb.Len()
	if n > m {
		n = m
	}
	data := lb.buf[:n]
	lb.buf = lb.buf[n:]
	return data
}

func (lb *LazyBuffer) tryGrowByReslice(n int) (int, bool) {
	if l := len(lb.buf); n <= cap(lb.buf)-l {
		lb.buf = lb.buf[:l+n]
		return l, true
	}
	return 0, false
}

func (lb *LazyBuffer) growSlice(n int) {
	// TODO(http://golang.org/issue/51462): We should rely on the append-make
	// pattern so that the compiler can call runtime.growslice. For example:
	//	return append(b, make([]byte, n)...)
	// This avoids unnecessary zero-ing of the first len(b) bytes of the
	// allocated slice, but this pattern causes b to escape onto the heap.
	//
	// Instead use the append-make pattern with a nil slice to ensure that
	// we allocate buffers rounded up to the closest size class.
	c := len(lb.buf) + n // ensure enough space for n elements
	if c < 2*cap(*lb.ref) {
		// The growth rate has historically always been 2x. In the future,
		// we could rely purely on append to determine the growth rate.
		c = 2 * cap(*lb.ref)
	}
	b2 := append([]byte(nil), make([]byte, c)...)
	copy(b2, lb.buf)
	lb.buf = b2
	lb.ref = &b2
}

func (lb *LazyBuffer) grow(n int) int {
	m := lb.Len()
	// Try to grow by means of a reslice.
	if i, ok := lb.tryGrowByReslice(n); ok {
		return i
	}

	if lb.ref == nil {
		lb.ref = bytePool.Get().(*[]byte)
	}

	c := cap(*lb.ref)
	if n <= c/2-m {
		// We can slide things down instead of allocating a new
		// slice. We only need m+n <= c to slide, but
		// we instead let capacity get twice as large so we
		// don't spend all our time copying.
		copy(*lb.ref, lb.buf)
	} else {
		// Add b.off to account for b.buf[:b.off] being sliced off the front.
		lb.growSlice(n)
	}

	lb.buf = (*lb.ref)[:m+n]
	return m
}

func (lb *LazyBuffer) Grow(n int) {
	m := lb.grow(n)
	lb.buf = lb.buf[:m]
}

func (lb *LazyBuffer) Extend(n int) {
	lb.grow(n)
}

func (lb *LazyBuffer) Truncate(n int) {
	if 0 <= n && n <= len(lb.buf) {
		lb.buf = lb.buf[:n]
	}
}

func (lb *LazyBuffer) Write(p []byte) (n int, err error) {
	m := len(lb.buf)
	if lb.ref == nil {
		oldData := lb.buf
		lb.ref = bytePool.Get().(*[]byte)
		lb.buf = (*lb.ref)[:0]
		lb.grow(m + len(p))
		copy(lb.buf, oldData)
	} else {
		lb.grow(len(p))
	}

	return copy(lb.buf[m:], p), nil
}

func (lb *LazyBuffer) Set(p []byte) {
	if lb.ref == nil && lb.Len() == 0 {
		lb.buf = p
	} else {
		lb.Write(p)
	}
}

func (lb *LazyBuffer) Done() {
	lb.buf = nil
	if lb.ref != nil {
		bytePool.Put(lb.ref)
		lb.ref = nil
	}
}

func (lb *LazyBuffer) IsLazy() bool {
	return lb.ref == nil
}
