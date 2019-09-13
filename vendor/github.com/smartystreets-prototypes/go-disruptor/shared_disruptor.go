package disruptor

type SharedDisruptor struct {
	writer  *SharedWriter
	readers []*Reader
}

func (this SharedDisruptor) Writer() *SharedWriter {
	return this.writer
}

func (this SharedDisruptor) Start() {
	for _, item := range this.readers {
		item.Start()
	}
}

func (this SharedDisruptor) Stop() {
	for _, item := range this.readers {
		item.Stop()
	}
}
