// +build freebsd dragonfly darwin
// +build !poll_opt

package netpoll

import "golang.org/x/sys/unix"

type epollevent = unix.Kevent_t
