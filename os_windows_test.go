//go:build windows

package gnet

import (
	"syscall"
)

func SysClose(fd int) error {
	return syscall.CloseHandle(syscall.Handle(fd))
}
