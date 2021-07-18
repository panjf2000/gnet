// Created by cgo -cdefs and converted (by hand) to Go
// ../cmd/cgo/cgo -cdefs defs_linux.go defs1_linux.go defs2_linux.go

// +build poll_opt

package netpoll

type epollevent struct {
	events uint32
	_pad   uint32
	data   [8]byte // to match amd64
}
