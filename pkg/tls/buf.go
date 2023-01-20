package tls

import (
	"io"
	"unsafe"
)

type MsgBuffer struct {
	b []byte
	l int // Total length of buffered data
	i int // Position of unread buffered data
}

const (
	blockSize   = 8192 // clean up the data when i >= blocksize
	appendSize  = 4096
	defaultSize = 4096
)

// New returns a new MsgBuffer whose buffer has the given size.
func NewMsgBuffer(n int) *MsgBuffer {
	return &MsgBuffer{b: make([]byte, 0, n)}
}

func (w *MsgBuffer) Reset() {
	w.l = 0
	w.i = 0
}

// clean up the data when i >= blockSize
func (w *MsgBuffer) clean() {
	if w.i >= blockSize {
		copy(w.b[:w.l-w.i], w.b[w.i:w.l])
		w.l -= w.i
		w.i = 0
	}
}

// grow the buffer size if the size of current buffer cannot fit the new incoming data.
func (w *MsgBuffer) grow() {
	if len(w.b) < w.l {
		if cap(w.b) < w.l {
			add := w.l - len(w.b)
			if add > appendSize {
				w.b = append(w.b, make([]byte, add)...)
			} else {
				w.b = append(w.b, make([]byte, appendSize)...)
			}
		}
		w.b = w.b[:w.l]
	}
}

func (w *MsgBuffer) Make(l int) []byte {
	w.clean()
	o := w.l
	w.l += l
	w.grow()
	return w.b[o:w.l]
}

func (w *MsgBuffer) Write(b []byte) (int, error) {
	w.clean()
	l := len(b)
	o := w.l
	w.l += l
	w.grow()
	copy(w.b[o:w.l], b)
	return l, nil
}

func (w *MsgBuffer) WriteString(s string) {
	w.clean()
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	b := *(*[]byte)(unsafe.Pointer(&h))
	l := len(b)
	o := w.l
	w.l += l
	w.grow()
	copy(w.b[o:w.l], b)
}

func (w *MsgBuffer) WriteByte(s byte) error {
	w.clean()
	w.l++
	w.grow()
	w.b[w.l-1] = s

	return nil
}

func (w *MsgBuffer) Bytes() []byte {
	return w.b[w.i:w.l]
}

func (w *MsgBuffer) Peek(n int) []byte {
	end := w.i + n
	if end > w.l {
		end = w.l
	}
	return w.b[w.i:end]
}

func (w *MsgBuffer) Len() int {
	return w.l - w.i
}

func (w *MsgBuffer) Truncate(i int) {
	l := w.i + i
	if l < w.l {
		w.l = l
	}
}

func (w *MsgBuffer) String() string {
	b := make([]byte, w.l-w.i)
	copy(b, w.b[w.i:w.l])
	return *(*string)(unsafe.Pointer(&b))
}

// Discard skips the next n bytes by advancing the read pointer.
func (r *MsgBuffer) Discard(l int) {
	if l <= 0 {
		return
	}
	if l < r.Len() {
		r.i += l
	} else {
		r.Reset()
	}
}

func (r *MsgBuffer) Close() error {
	return nil
}

func (r *MsgBuffer) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.i == r.l {
		return 0, io.EOF
	}
	o := r.i
	r.i += len(p)
	if r.i > r.l {
		r.i = r.l
	}
	copy(p, r.b[o:r.i])
	return r.i - o, nil
}

// ReadByte reads and returns the next byte from the input or ErrIsEmpty.
func (r *MsgBuffer) ReadByte() (b byte, err error) {
	if r.i == r.l {
		return 0, io.EOF
	}
	b = r.b[r.i]
	r.i++
	return b, err
}

// IsEmpty tells if this MsgBuffer is empty.
func (b *MsgBuffer) IsEmpty() bool {
	return b.i == b.l
}
