package byteslice

import (
	"bytes"
	"fmt"
	"testing"
)

func TestByteSlice(t *testing.T) {
	var pool Pool
	for _, api := range []struct {
		name string
		get  func(int) []byte
		put  func([]byte)
	}{
		{"builtin", Get, Put},
		{"zero-value", pool.Get, pool.Put},
	} {
		t.Run(api.name, func(t *testing.T) {
			buf := api.get(8)
			if err := byteSliceShapeError(buf, 8); err != nil {
				t.Fatal(err)
			}
			copy(buf, "ff")
			if string(buf[:2]) != "ff" {
				t.Fatal("expect copy result is ff, but not")
			}
			api.put(buf)
			// Put transfers ownership. Get(7) may reuse this array or
			// allocate another; neither identity nor old contents is promised.
			for _, size := range []int{7, -1, 0, 1, 2, 3, 8, 9, 15, 16, 17, 1023, 1024, 1025} {
				t.Run(fmt.Sprint(size), func(t *testing.T) {
					first := api.get(size)
					if err := byteSliceShapeError(first, size); err != nil {
						t.Fatal(err)
					}
					defer api.put(first)
					if size <= 0 {
						return
					}
					second := api.get(size)
					if err := byteSlicePairError(first, second, size); err != nil {
						t.Fatal(err)
					}
					api.put(second)
				})
			}
			// Accept nil, zero-capacity, and non-power-of-two capacity
			// buffers, including a zero-length view with spare capacity.
			for _, returned := range [][]byte{nil, {}, make([]byte, 3), make([]byte, 0, 9)} {
				api.put(returned)
			}
			for _, size := range []int{1, 2, 3, 7, 8, 9} {
				first, second := api.get(size), api.get(size)
				if err := byteSlicePairError(first, second, size); err != nil {
					t.Fatal(err)
				}
				api.put(first)
				api.put(second)
			}
		})
	}
}

func byteSliceShapeError(buf []byte, size int) error {
	if size <= 0 {
		if buf != nil {
			return fmt.Errorf("Get(%d) returned non-nil slice", size)
		}
		return nil
	}
	if len(buf) != size || cap(buf) < size {
		return fmt.Errorf("Get(%d): len=%d cap=%d", size, len(buf), cap(buf))
	}
	return nil
}

func byteSlicePairError(first, second []byte, size int) error {
	if err := byteSliceShapeError(first, size); err != nil {
		return err
	}
	if err := byteSliceShapeError(second, size); err != nil {
		return err
	}
	wantFirst := bytes.Repeat([]byte{0xa5}, size)
	wantSecond := bytes.Repeat([]byte{0x5a}, size)
	copy(first, wantFirst)
	copy(second, wantSecond)
	if !bytes.Equal(first, wantFirst) || !bytes.Equal(second, wantSecond) {
		return fmt.Errorf("Get(%d): writes to live buffers are not independent", size)
	}
	return nil
}

func TestByteSliceOracles(t *testing.T) {
	shared := make([]byte, 9)
	for _, tc := range []struct {
		name          string
		first, second []byte
		wantErr       bool
	}{
		{"independent", make([]byte, 8), make([]byte, 8), false},
		{"short", make([]byte, 7), make([]byte, 8), true},
		{"long", make([]byte, 9), make([]byte, 8), true},
		{"nil", nil, make([]byte, 8), true},
		{"second-short", make([]byte, 8), make([]byte, 7), true},
		{"same-array", shared[:8], shared[:8], true},
		{"overlap", shared[:8], shared[1:], true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := byteSlicePairError(tc.first, tc.second, 8); (err != nil) != tc.wantErr {
				t.Fatalf("pair oracle: error=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
	for _, size := range []int{-1, 0} {
		if err := byteSliceShapeError(nil, size); err != nil {
			t.Fatal(err)
		}
		if err := byteSliceShapeError([]byte{}, size); err == nil {
			t.Fatalf("shape oracle accepted non-nil slice for size %d", size)
		}
	}
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
