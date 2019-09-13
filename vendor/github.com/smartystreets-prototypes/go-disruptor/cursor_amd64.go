package disruptor

func (this *Cursor) Store(sequence int64) {
	this.sequence = sequence
}

func (this *Cursor) Load() int64 {
	return this.sequence
}

func (this *Cursor) Read(noop int64) int64 {
	return this.sequence
}
