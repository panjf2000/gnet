// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package gnet

// LoadBalancing represents the the type of load-balancing algorithm.
type LoadBalancing int

const (
	// RoundRobin assigns the next accepted connection to the event-loop by polling event-loop list.
	RoundRobin LoadBalancing = iota

	// LeastConnections assigns the next accepted connection to the event-loop that is
	// serving the least number of active connections at the current time.
	LeastConnections

	// SourceAddrHash assignes the next accepted connection to the event-loop by hashing socket fd.
	SourceAddrHash
)

// IEventLoopGroup represents a set of event-loops.
type (
	IEventLoopGroup interface {
		register(*eventloop)
		next(int) *eventloop
		iterate(func(int, *eventloop) bool)
		len() int
	}

	// roundRobinEventLoopGroup with RoundRobin algorithm.
	roundRobinEventLoopGroup struct {
		nextLoopIndex int
		eventLoops    []*eventloop
		size          int
	}

	// leastConnectionsEventLoopGroup with Least-Connections algorithm.
	leastConnectionsEventLoopGroup []*eventloop

	// sourceAddrHashEventLoopGroup with Hash algorithm.
	sourceAddrHashEventLoopGroup struct {
		eventLoops []*eventloop
		size       int
	}
)

func (g *roundRobinEventLoopGroup) register(el *eventloop) {
	g.eventLoops = append(g.eventLoops, el)
	g.size++
}

// next returns the eligible event-loop based on Round-Robin algorithm.
func (g *roundRobinEventLoopGroup) next(_ int) (el *eventloop) {
	el = g.eventLoops[g.nextLoopIndex]
	if g.nextLoopIndex++; g.nextLoopIndex >= g.size {
		g.nextLoopIndex = 0
	}
	return
}

func (g *roundRobinEventLoopGroup) iterate(f func(int, *eventloop) bool) {
	for i, el := range g.eventLoops {
		if !f(i, el) {
			break
		}
	}
}

func (g *roundRobinEventLoopGroup) len() int {
	return g.size
}

func (g *leastConnectionsEventLoopGroup) register(el *eventloop) {
	*g = append(*g, el)
}

// next returns the eligible event-loop based on least-connections algorithm.
func (g *leastConnectionsEventLoopGroup) next(_ int) (el *eventloop) {
	eventLoops := *g
	el = eventLoops[0]
	leastConnCount := el.loadConnCount()
	var (
		curEventLoop *eventloop
		curConnCount int32
	)
	for _, curEventLoop = range eventLoops[1:] {
		if curConnCount = curEventLoop.loadConnCount(); curConnCount < leastConnCount {
			leastConnCount = curConnCount
			el = curEventLoop
		}
	}
	return
}

func (g *leastConnectionsEventLoopGroup) iterate(f func(int, *eventloop) bool) {
	eventLoops := *g
	for i, el := range eventLoops {
		if !f(i, el) {
			break
		}
	}
}

func (g *leastConnectionsEventLoopGroup) len() int {
	return len(*g)
}

func (g *sourceAddrHashEventLoopGroup) register(el *eventloop) {
	g.eventLoops = append(g.eventLoops, el)
	g.size++
}

// next returns the eligible event-loop by taking the remainder of a given fd as the index of event-loop list.
func (g *sourceAddrHashEventLoopGroup) next(hashCode int) *eventloop {
	return g.eventLoops[hashCode%g.size]
}

func (g *sourceAddrHashEventLoopGroup) iterate(f func(int, *eventloop) bool) {
	for i, el := range g.eventLoops {
		if !f(i, el) {
			break
		}
	}
}

func (g *sourceAddrHashEventLoopGroup) len() int {
	return g.size
}
