// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package gnet

// LoadBalance sets the load balancing method.
//type LoadBalance int
//
//const (
//	// RoundRobin requests that connections are distributed to a loop in a
//	// round-robin fashion.
//	RoundRobin LoadBalance = iota
//	// Random requests that connections are randomly distributed.
//	Random
//	// LeastConnections assigns the next accepted connection to the loop with
//	// the least number of active connections.
//	LeastConnections
//)

// IEventLoopGroup ...
type IEventLoopGroup interface {
	register(*loop)
	next() *loop
	iterate(func(int, *loop) bool)
	len() int
}

type eventLoopGroup struct {
	//loadBalance   LoadBalance
	nextLoopIndex int
	eventLoops    []*loop
	size          int
}

func (g *eventLoopGroup) register(lp *loop) {
	g.eventLoops = append(g.eventLoops, lp)
	g.size++
}

// Built-in load-balance algorithm is Round-Robin.
// TODO: support more load-balance algorithms.
func (g *eventLoopGroup) next() (lp *loop) {
	//return g.nextByRoundRobin()
	lp = g.eventLoops[g.nextLoopIndex]
	g.nextLoopIndex++
	if g.nextLoopIndex >= g.size {
		g.nextLoopIndex = 0
	}
	return
}

//func (g *eventLoopGroup) nextByRoundRobin() (lp *loop) {
//	lp = g.eventLoops[g.nextLoopIndex]
//	g.nextLoopIndex++
//	if g.nextLoopIndex >= g.size {
//		g.nextLoopIndex = 0
//	}
//	return
//}

func (g *eventLoopGroup) iterate(f func(int, *loop) bool) {
	for i, lp := range g.eventLoops {
		if !f(i, lp) {
			break
		}
	}
}

func (g *eventLoopGroup) len() int {
	return g.size
}
