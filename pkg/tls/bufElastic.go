package tls

import (
	"io"
	"sync"
)

// EMsgBuffer is the elastic wrapper of EMsgBuffer.
type EMsgBuffer struct {
	mb *MsgBuffer
}

var msgBufferPool = sync.Pool{
	New: func() any {
		return NewMsgBuffer(defaultSize)
	},
}

func (b *EMsgBuffer) instance() *MsgBuffer {
	if b.mb == nil {
		b.mb = msgBufferPool.New().(*MsgBuffer)
	}
	return b.mb
}

// Done checks and returns the internal MsgBuffer to pool.
func (b *EMsgBuffer) Done() {
	if b.mb != nil {
		b.mb.Reset()
		msgBufferPool.Put(b.mb)
		b.mb = nil
	}
}

func (b *EMsgBuffer) DoneIfEmpty() {
	b.done()
}

func (b *EMsgBuffer) done() {
	if b.mb != nil && b.mb.IsEmpty() {
		b.mb.Reset()
		msgBufferPool.Put(b.mb)
		b.mb = nil
	}
}

func (b *EMsgBuffer) Make(l int) []byte {
	return b.instance().Make(l)
}

// Write writes len(p) bytes from p to the underlying buf.
func (b *EMsgBuffer) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return b.instance().Write(p)
}

// WriteString writes the contents of the string s to buffer, which accepts a slice of bytes.
func (b *EMsgBuffer) WriteString(s string) {
	b.instance().WriteString(s)
}

// WriteByte writes one byte into buffer.
func (b *EMsgBuffer) WriteByte(c byte) error {
	return b.instance().WriteByte(c)
}

// Bytes returns all available read bytes. It does not move the read pointer and only copy the available data.
func (b *EMsgBuffer) Bytes() []byte {
	if b.mb == nil {
		return nil
	}
	return b.mb.Bytes()
}

// Bytes returns first n readable bytes. It does not move the read pointer and only copy the available data.
func (b *EMsgBuffer) Peek(n int) []byte {
	if b.mb == nil {
		return nil
	}
	return b.mb.Peek(n)
}

// Len returns the length of the underlying buffer.
func (b *EMsgBuffer) Len() int {
	if b.mb == nil {
		return 0
	}
	return b.mb.Len()
}

// truncate the total number of readable bytes to i
func (b *EMsgBuffer) Truncate(i int) {
	if b.mb != nil {
		b.mb.Truncate(i)
		b.done()
	}
}

func (b *EMsgBuffer) String() string {
	if b.mb == nil {
		return ""
	}
	return b.mb.String()
}

// Discard skips the next n bytes by advancing the read pointer.
func (b *EMsgBuffer) Discard(l int) {
	if b.mb != nil {
		b.mb.Discard(l)
		b.done()
	}
}

// Discard skips the next n bytes by advancing the read pointer, but holding the MsgBuffer temporarily.
// Doing so can ensure one can use the data return by Peek not used by another thread.
// Therefore, thread-safe is guaranteed.
func (b *EMsgBuffer) DiscardWithoutDone(l int) {
	if b.mb != nil {
		b.mb.Discard(l)
	}
}

func (b *EMsgBuffer) Close() error {
	if b.mb == nil {
		return nil
	}
	return b.Close()

}

func (b *EMsgBuffer) Read(p []byte) (n int, err error) {
	if b.mb == nil {
		return 0, io.EOF
	}
	defer b.done()
	return b.mb.Read(p)
}

// ReadByte reads and returns the next byte from the input or ErrIsEmpty.
func (b *EMsgBuffer) ReadByte() (byte, error) {
	if b.mb == nil {
		return 0, io.EOF
	}
	defer b.done()
	return b.mb.ReadByte()
}

// IsEmpty tells if this MsgBuffer is empty.
func (b *EMsgBuffer) IsEmpty() bool {
	if b.mb == nil {
		return true
	}
	return b.mb.IsEmpty()
}
