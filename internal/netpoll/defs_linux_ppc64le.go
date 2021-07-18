// created by cgo -cdefs and then converted to Go
// cgo -cdefs defs_linux.go defs3_linux.go

// +build poll_opt

package netpoll

type epollevent struct {
	events    uint32
	pad_cgo_0 [4]byte
	data      [8]byte // unaligned uintptr
}
