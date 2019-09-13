package disruptor

type Barrier interface {
	Read(int64) int64
}
