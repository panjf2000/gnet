package disruptor

type Disruptor struct {
	writer  *Writer
	readers []*Reader
}

func (this Disruptor) Writer() *Writer {
	return this.writer
}

func (this Disruptor) Start() {
	for _, item := range this.readers {
		item.Start()
	}
}

func (this Disruptor) Stop() {
	for _, item := range this.readers {
		item.Stop()
	}
}
