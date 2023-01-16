package tls

import (
	"io"
	"unsafe"
)

type MsgBuffer struct {
	b []byte
	l int //长度
	i int //起点位置
}

const (
	blocksize  = 1024 * 5 //清理失效数据阈值
	appendsize = 4096
)

func NewBuffer(n int) *MsgBuffer {
	return &MsgBuffer{b: make([]byte, 0, n)}
}

func (w *MsgBuffer) Reset() {
	w.l = 0
	w.i = 0
}

func (w *MsgBuffer) Make(l int) []byte {
	if w.i > blocksize {
		copy(w.b[:w.l-w.i], w.b[w.i:w.l])
		w.l -= w.i
		w.i = 0
	}
	o := w.l
	w.l += l
	if len(w.b) < w.l { //扩容
		if cap(w.b) < w.l {
			add := w.l - len(w.b)
			if add > appendsize {
				w.b = append(w.b, make([]byte, add)...)
			} else {
				w.b = append(w.b, make([]byte, appendsize)...)
			}
		}
		w.b = w.b[:w.l]
	}
	return w.b[o:w.l]
}

func (w *MsgBuffer) Write(b []byte) (int, error) {
	if w.i > blocksize {
		copy(w.b[:w.l-w.i], w.b[w.i:w.l])
		w.l -= w.i
		w.i = 0
	}
	l := len(b)
	o := w.l
	w.l += l
	if len(w.b) < w.l {
		if cap(w.b) < w.l {
			add := w.l - len(w.b)
			if add > appendsize {
				w.b = append(w.b, make([]byte, add)...)
			} else {
				w.b = append(w.b, make([]byte, appendsize)...)
			}
		}
		w.b = w.b[:w.l]
	}
	copy(w.b[o:w.l], b)
	return l, nil
}

func (w *MsgBuffer) WriteString(s string) {
	if w.i > blocksize {
		copy(w.b[:w.l-w.i], w.b[w.i:w.l])
		w.l -= w.i
		w.i = 0
	}
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	b := *(*[]byte)(unsafe.Pointer(&h))
	l := len(b)
	o := w.l
	w.l += l
	if len(w.b) < w.l { //扩容
		if cap(w.b) < w.l {
			add := w.l - len(w.b)
			if add > appendsize {
				w.b = append(w.b, make([]byte, add)...)
			} else {
				w.b = append(w.b, make([]byte, appendsize)...)
			}
		}
		w.b = w.b[:w.l]
	}
	copy(w.b[o:w.l], b)
}

func (w *MsgBuffer) WriteByte(s byte) error {
	if w.i > blocksize {
		copy(w.b[:w.l-w.i], w.b[w.i:w.l])
		w.l -= w.i
		w.i = 0
	}
	w.l++
	if len(w.b) < w.l {
		if cap(w.b) < w.l {
			add := w.l - len(w.b)
			if add > appendsize {
				w.b = append(w.b, make([]byte, add)...)
			} else {
				w.b = append(w.b, make([]byte, appendsize)...)
			}
		}
		w.b = w.b[:w.l]
	}
	w.b[w.l-1] = s

	return nil
}

func (w *MsgBuffer) Bytes() []byte {
	return w.b[w.i:w.l]
}

func (w *MsgBuffer) PreBytes(n int) []byte {
	end := w.i + n
	if end > w.l {
		end = w.l
	}
	return w.b[w.i:end]
}

func (w *MsgBuffer) Len() int {
	return w.l - w.i
}

func (w *MsgBuffer) Next(l int) []byte {
	o := w.i
	w.i += l
	if w.i > w.l {
		w.i = w.l
	}
	return w.b[o:w.i]
}

func (w *MsgBuffer) Truncate(i int) {
	w.l = w.i + i
}

func (w *MsgBuffer) String() string {
	b := make([]byte, w.l-w.i)
	copy(b, w.b[w.i:w.l])
	return *(*string)(unsafe.Pointer(&b))
}

// New returns a new MsgBuffer whose buffer has the given size.
func New(size int) *MsgBuffer {

	return &MsgBuffer{
		b: make([]byte, size),
	}
}

// Shift shifts the "read" pointer.
func (r *MsgBuffer) Shift(len int) {
	if len <= 0 {
		return
	}
	if len < r.Len() {
		r.i += len
		if r.i > r.l {
			r.i = r.l
		}
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
