package disruptor

// Cursors should be a party of the same backing array to keep them as close together as possible:
// https://news.ycombinator.com/item?id=7800825
type (
	Wireup struct {
		capacity int64
		groups   [][]Consumer
		cursors  []*Cursor // backing array keeps cursors (with padding) in contiguous memory
	}
)

func Configure(capacity int64) Wireup {
	return Wireup{
		capacity: capacity,
		groups:   [][]Consumer{},
		cursors:  []*Cursor{NewCursor()},
	}
}

func (this Wireup) WithConsumerGroup(consumers ...Consumer) Wireup {
	if len(consumers) == 0 {
		return this
	}

	target := make([]Consumer, len(consumers))
	copy(target, consumers)

	for i := 0; i < len(consumers); i++ {
		this.cursors = append(this.cursors, NewCursor())
	}

	this.groups = append(this.groups, target)
	return this
}

func (this Wireup) Build() Disruptor {
	allReaders := []*Reader{}
	written := this.cursors[0]
	var upstream Barrier = this.cursors[0]
	cursorIndex := 1 // 0 index is reserved for the writer Cursor

	for groupIndex, group := range this.groups {
		groupReaders, groupBarrier := this.buildReaders(groupIndex, cursorIndex, written, upstream)
		for _, item := range groupReaders {
			allReaders = append(allReaders, item)
		}
		upstream = groupBarrier
		cursorIndex += len(group)
	}

	writer := NewWriter(written, upstream, this.capacity)
	return Disruptor{writer: writer, readers: allReaders}
}

func (this Wireup) BuildShared() SharedDisruptor {
	allReaders := []*Reader{}
	written := this.cursors[0]
	writerBarrier := NewSharedWriterBarrier(written, this.capacity)
	var upstream Barrier = writerBarrier
	cursorIndex := 1 // 0 index is reserved for the writer Cursor

	for groupIndex, group := range this.groups {
		groupReaders, groupBarrier := this.buildReaders(groupIndex, cursorIndex, written, upstream)
		for _, item := range groupReaders {
			allReaders = append(allReaders, item)
		}
		upstream = groupBarrier
		cursorIndex += len(group)
	}

	writer := NewSharedWriter(writerBarrier, upstream)
	return SharedDisruptor{writer: writer, readers: allReaders}
}

func (this Wireup) buildReaders(consumerIndex, cursorIndex int, written *Cursor, upstream Barrier) ([]*Reader, Barrier) {
	barrierCursors := []*Cursor{}
	readers := []*Reader{}

	for _, consumer := range this.groups[consumerIndex] {
		cursor := this.cursors[cursorIndex]
		barrierCursors = append(barrierCursors, cursor)
		reader := NewReader(cursor, written, upstream, consumer)
		readers = append(readers, reader)
		cursorIndex++
	}

	if len(this.groups[consumerIndex]) == 1 {
		return readers, barrierCursors[0]
	} else {
		return readers, NewCompositeBarrier(barrierCursors...)
	}
}
