package disruptor

type CompositeBarrier []*Cursor

func NewCompositeBarrier(upstream ...*Cursor) CompositeBarrier {
	if len(upstream) == 0 {
		panic("At least one upstream cursor is required.")
	}

	cursors := make([]*Cursor, len(upstream))
	copy(cursors, upstream)
	return CompositeBarrier(cursors)
}

func (this CompositeBarrier) Read(noop int64) int64 {
	minimum := MaxSequenceValue
	for _, item := range this {
		sequence := item.Load()
		if sequence < minimum {
			minimum = sequence
		}
	}

	return minimum
}
