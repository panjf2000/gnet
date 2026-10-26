//go:build (darwin || dragonfly || freebsd || linux || netbsd || openbsd) && !poll_opt

package netpoll_test

import "github.com/panjf2000/gnet/v2/pkg/netpoll"

func pr766ExamplePolling(p *netpoll.Poller, callback netpoll.PollEventHandler) error {
	return p.Polling(callback)
}
