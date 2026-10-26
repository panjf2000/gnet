//go:build linux && poll_opt

package netpoll

func pr766LifecyclePoll(p *Poller) error { return p.Polling() }

func pr766LifecycleWakeFD(p *Poller) int { return p.epa.FD }
