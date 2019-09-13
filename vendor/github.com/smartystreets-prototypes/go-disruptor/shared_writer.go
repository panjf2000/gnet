package disruptor

import (
	"runtime"
	"sync/atomic"
)

type SharedWriter struct {
	written   *Cursor
	upstream  Barrier
	capacity  int64
	gate      *Cursor
	mask      int64
	shift     uint8
	committed []int32
}

func NewSharedWriter(write *SharedWriterBarrier, upstream Barrier) *SharedWriter {
	return &SharedWriter{
		written:   write.written,
		upstream:  upstream,
		capacity:  write.capacity,
		gate:      NewCursor(),
		mask:      write.mask,
		shift:     write.shift,
		committed: write.committed,
	}
}

func (this *SharedWriter) Reserve(count int64) int64 {
	for {
		previous := this.written.Load()
		upper := previous + count

		for spin := int64(0); upper-this.capacity > this.gate.Load(); spin++ {
			if spin&SpinMask == 0 {
				runtime.Gosched() // LockSupport.parkNanos(1L); http://bit.ly/1xiDINZ
			}
			this.gate.Store(this.upstream.Read(0))
		}

		if atomic.CompareAndSwapInt64(&this.written.sequence, previous, upper) {
			return upper
		}
	}
}

func (this *SharedWriter) Commit(lower, upper int64) {
	if lower > upper {
		panic("Attempting to commit a sequence where the lower reservation is greater than the higher reservation.")
	} else if (upper - lower) > this.capacity {
		panic("Attempting to commit a reservation larger than the size of the ring buffer. (upper-lower > this.capacity)")
	} else if lower == upper {
		this.committed[upper&this.mask] = int32(upper >> this.shift)
	} else {
		// working down the array rather than up keeps all items in the commit together
		// otherwise the reader(s) could split up the group
		for upper >= lower {
			this.committed[upper&this.mask] = int32(upper >> this.shift)
			upper--
		}

	}
}
