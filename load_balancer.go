// Copyright (c) 2019 Andy Pan
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package gnet

import (
	"container/heap"
	"hash/crc32"
	"math"
	"net"

	"github.com/panjf2000/gnet/internal"
)

// LoadBalancing represents the the type of load-balancing algorithm.
type LoadBalancing int

const (
	// RoundRobin assigns the next accepted connection to the event-loop by polling event-loop list.
	RoundRobin LoadBalancing = iota

	// LeastConnections assigns the next accepted connection to the event-loop that is
	// serving the least number of active connections at the current time.
	LeastConnections

	// SourceAddrHash assignes the next accepted connection to the event-loop by hashing the remote address.
	SourceAddrHash
)

type (
	// loadBalancer is a interface which manipulates the event-loop set.
	loadBalancer interface {
		register(*eventloop)
		next(net.Addr) *eventloop
		iterate(func(int, *eventloop) bool)
		len() int
	}

	// roundRobinLoadBalancer with Round-Robin algorithm.
	roundRobinLoadBalancer struct {
		nextLoopIndex int
		eventLoops    []*eventloop
		size          int
	}

	// leastConnectionsLoadBalancer with Least-Connections algorithm.
	leastConnectionsLoadBalancer struct {
		cachedRoot     *eventloop
		minHeap        minEventLoopHeap
		eventLoopsCopy []*eventloop
		size           int
	}

	// sourceAddrHashLoadBalancer with Hash algorithm.
	sourceAddrHashLoadBalancer struct {
		eventLoops []*eventloop
		size       int
	}
)

// ==================================== Implementation of Round-Robin load-balancer ====================================

func (lb *roundRobinLoadBalancer) register(el *eventloop) {
	el.idx = lb.size
	lb.eventLoops = append(lb.eventLoops, el)
	lb.size++
}

// next returns the eligible event-loop based on Round-Robin algorithm.
func (lb *roundRobinLoadBalancer) next(_ net.Addr) (el *eventloop) {
	el = lb.eventLoops[lb.nextLoopIndex]
	if lb.nextLoopIndex++; lb.nextLoopIndex >= lb.size {
		lb.nextLoopIndex = 0
	}
	return
}

func (lb *roundRobinLoadBalancer) iterate(f func(int, *eventloop) bool) {
	for i, el := range lb.eventLoops {
		if !f(i, el) {
			break
		}
	}
}

func (lb *roundRobinLoadBalancer) len() int {
	return lb.size
}

// ================================= Implementation of Least-Connections load-balancer =================================

// Leverage min-heap to optimize Least-Connections load-balancing.
type minEventLoopHeap []*eventloop

// Implement heap.Interface: Len, Less, Swap, Push, Pop.
func (h minEventLoopHeap) Len() int {
	return len(h)
}

func (h minEventLoopHeap) Less(i, j int) bool {
	return h[i].loadConn() < h[j].loadConn()
}

func (h minEventLoopHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].idx, h[j].idx = i, j
}

func (h *minEventLoopHeap) Push(x interface{}) {
	el := x.(*eventloop)
	el.idx = len(*h)
	*h = append(*h, el)
}

func (h *minEventLoopHeap) Pop() interface{} {
	old := *h
	i := len(old) - 1
	x := old[i]
	old[i] = nil // avoid memory leak
	x.idx = -1   // for safety
	*h = old[:i]
	return x
}

func (lb *leastConnectionsLoadBalancer) register(el *eventloop) {
	heap.Push(&lb.minHeap, el)
	if el.idx == 0 {
		lb.cachedRoot = el
	}
	lb.eventLoopsCopy = append(lb.eventLoopsCopy, el)
	lb.size++
}

// standardDeviation calculates and returns the standard deviation of all connection numbers in event-loops.
func (lb *leastConnectionsLoadBalancer) standardDeviation() float64 {
	var sum, variance float64
	for _, el := range lb.minHeap {
		sum += float64(el.loadConn())
	}
	length := float64(lb.minHeap.Len())
	mean := sum / length
	for _, el := range lb.minHeap {
		diff := float64(el.loadConn()) - mean
		variance += diff * diff
	}
	variance /= length
	return math.Sqrt(variance)
}

// next returns the eligible event-loop by taking the root node from minimum heap based on Least-Connections algorithm.
func (lb *leastConnectionsLoadBalancer) next(_ net.Addr) (el *eventloop) {
	// In most cases, `next` method returns the cached event-loop immediately,
	// we don't readjust the heap every time a connection is created or closed,
	// but reconstruct the minimum heap only if the standard deviation of all
	// connection-numbers in event-loops is greater than the magic number,
	// to avoid introducing a global lock.
	if lb.standardDeviation() > 2.5 {
		// We choose 2.5 as the magic number here, which approximately restricts the difference value
		// between connection numbers in all event-loops to 10 ~ 20, on 8/16/32/48-core processors.
		heap.Init(&lb.minHeap)
		lb.cachedRoot = lb.minHeap[0]
	}
	return lb.cachedRoot
}

func (lb *leastConnectionsLoadBalancer) iterate(f func(int, *eventloop) bool) {
	for i, el := range lb.eventLoopsCopy {
		if !f(i, el) {
			break
		}
	}
}

func (lb *leastConnectionsLoadBalancer) len() int {
	return lb.size
}

// ======================================= Implementation of Hash load-balancer ========================================

func (lb *sourceAddrHashLoadBalancer) register(el *eventloop) {
	el.idx = lb.size
	lb.eventLoops = append(lb.eventLoops, el)
	lb.size++
}

// hash hashes a string to a unique hash code.
func (lb *sourceAddrHashLoadBalancer) hash(s string) int {
	v := int(crc32.ChecksumIEEE(internal.StringToBytes(s)))
	if v >= 0 {
		return v
	}
	return -v
}

// next returns the eligible event-loop by taking the remainder of a hash code as the index of event-loop list.
func (lb *sourceAddrHashLoadBalancer) next(netAddr net.Addr) *eventloop {
	hashCode := lb.hash(netAddr.String())
	return lb.eventLoops[hashCode%lb.size]
}

func (lb *sourceAddrHashLoadBalancer) iterate(f func(int, *eventloop) bool) {
	for i, el := range lb.eventLoops {
		if !f(i, el) {
			break
		}
	}
}

func (lb *sourceAddrHashLoadBalancer) len() int {
	return lb.size
}
