// Generated using cgo, then manually converted into appropriate naming and code
// for the Go runtime.
// go tool cgo -godefs defs_linux.go defs1_linux.go defs2_linux.go

// +build poll_opt

package netpoll

type epollevent struct {
	events    uint32
	pad_cgo_0 [4]byte
	data      [8]byte // unaligned uintptr
}
