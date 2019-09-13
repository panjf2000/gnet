package disruptor

const (
	MaxSequenceValue     int64 = (1 << 63) - 1
	InitialSequenceValue int64 = -1
	cpuCacheLinePadding        = 7
)

// TODO: research aligned read/write:
// https://github.com/fmstephe/fatomic/blob/master/slice.go
// https://groups.google.com/forum/#!topic/golang-nuts/XDfQUn4U_g8
// https://gist.github.com/anachronistic/7495541
// http://blog.chewxy.com/2013/12/10/pointer-tagging-in-go/
// http://www.goinggo.net/2014/01/concurrency-goroutines-and-gomaxprocs.html

type Cursor struct {
	sequence int64
	padding  [cpuCacheLinePadding]int64
}

func NewCursor() *Cursor {
	return &Cursor{sequence: InitialSequenceValue}
}
