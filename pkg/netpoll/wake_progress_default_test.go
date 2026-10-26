//go:build linux && !poll_opt

package netpoll

func pr766WakeSetFD(p *Poller, fd int) { p.efd = fd }
