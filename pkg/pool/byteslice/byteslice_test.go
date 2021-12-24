package byteslice

import (
	"runtime/debug"
	"testing"
)

func TestByteSlice(t *testing.T) {
	buf := Get(8)
	copy(buf, "ff")
	if string(buf[:2]) != "ff" {
		t.Fatal("expect copy result is ff, but not")
	}

	// Disable GC to test re-acquire the same data
	gc := debug.SetGCPercent(-1)

	Put(buf)

	newBuf := Get(7)
	if &newBuf[0] != &buf[0] {
		t.Fatal("expect newBuf and buf to be the same array")
	}
	if string(newBuf[:2]) != "ff" {
		t.Fatal("expect the newBuf is the buf, but not")
	}

	// Re-enable GC
	debug.SetGCPercent(gc)
}

func BenchmarkByteSlice(b *testing.B) {
	b.Run("Run.N", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			bs := Get(1024)
			Put(bs)
		}
	})
	b.Run("Run.Parallel", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				bs := Get(1024)
				Put(bs)
			}
		})
	})
}
