package disruptor

func (this *Writer) Commit(lower, upper int64) {
	this.written.Store(upper)
}
