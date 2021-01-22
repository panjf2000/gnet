package internal

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

type originSpinLock uint32

func (sl *originSpinLock) Lock() {
	for !atomic.CompareAndSwapUint32((*uint32)(sl), 0, 1) {
		runtime.Gosched()
	}
}

func (sl *originSpinLock) Unlock() {
	atomic.StoreUint32((*uint32)(sl), 0)
}

func OriginSpinLock() sync.Locker {
	return new(originSpinLock)
}

type backOffSpinLock uint32

func (sl *backOffSpinLock) Lock() {
	wait := 1
	for !atomic.CompareAndSwapUint32((*uint32)(sl), 0, 1) {
		for i := 0; i < wait; i++ {
			runtime.Gosched()
		}
		wait <<= 1
	}
}

func (sl *backOffSpinLock) Unlock() {
	atomic.StoreUint32((*uint32)(sl), 0)
}

func BackOffSpinLock() sync.Locker {
	return new(backOffSpinLock)
}

func BenchmarkSpinLock(b *testing.B) {
	spin := OriginSpinLock()
	//b.SetParallelism(200)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			spin.Lock()
			_ = 1 + 1
			spin.Unlock()
		}
	})
}

func BenchmarkBackOffSpinLock(b *testing.B) {
	spin := BackOffSpinLock()
	//b.SetParallelism(200)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			spin.Lock()
			_ = 1 + 1
			spin.Unlock()
		}
	})
}
