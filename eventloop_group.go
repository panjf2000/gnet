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

// IEventLoopGroup represents a set of event-loops.
type (
	IEventLoopGroup interface {
		register(*eventloop)
		next() *eventloop
		iterate(func(int, *eventloop) bool)
		len() int
	}

	eventLoopGroup struct {
		nextLoopIndex int
		eventLoops    []*eventloop
		size          int
	}

	eventLoopPriorityGroup struct {
		eventLoops []*eventloop
		size       int
	}
)

func (g *eventLoopPriorityGroup) register(el *eventloop) {
	g.eventLoops = append(g.eventLoops, el)
	g.size++
}

// 常量级循环
func (g *eventLoopPriorityGroup) next() (el *eventloop) {
	var (
		minIdx = 0
	)
	for i := 1; i < g.size; i++ {
		if g.eventLoops[i].priority < g.eventLoops[minIdx].priority {
			minIdx = i
		}
	}
	g.eventLoops[minIdx].priority++
	return g.eventLoops[minIdx]
}
func (g *eventLoopPriorityGroup) iterate(f func(int, *eventloop) bool) {
	for i, el := range g.eventLoops {
		if !f(i, el) {
			break
		}
	}
}
func (g *eventLoopPriorityGroup) len() int {
	return g.size
}

func (g *eventLoopGroup) register(el *eventloop) {
	g.eventLoops = append(g.eventLoops, el)
	g.size++
}

// Built-in load-balance algorithm is Round-Robin.
// TODO: support more load-balance algorithms.
func (g *eventLoopGroup) next() (el *eventloop) {
	el = g.eventLoops[g.nextLoopIndex]
	if g.nextLoopIndex++; g.nextLoopIndex >= g.size {
		g.nextLoopIndex = 0
	}
	return
}

func (g *eventLoopGroup) iterate(f func(int, *eventloop) bool) {
	for i, el := range g.eventLoops {
		if !f(i, el) {
			break
		}
	}
}

func (g *eventLoopGroup) len() int {
	return g.size
}
