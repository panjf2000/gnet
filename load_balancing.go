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
	"sync"
	"sync/atomic"
)

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

type (
	// loadBalancer is a interface which manipulates the event-loop set.
	loadBalancer interface {
		register(*eventloop)
		next(int) *eventloop
		iterate(func(int, *eventloop) bool)
		len() int
		calibrate(*eventloop, int32)
	}

	// roundRobinEventLoopSet with Round-Robin algorithm.
	roundRobinEventLoopSet struct {
		nextLoopIndex int
		eventLoops    []*eventloop
		size          int
	}

	// leastConnectionsEventLoopSet with Least-Connections algorithm.
	leastConnectionsEventLoopSet struct {
		sync.RWMutex
		minHeap                 minEventLoopHeap
		cachedRoot              *eventloop
		threshold               int32
		calibrateConnsThreshold int32
	}

	// sourceAddrHashEventLoopSet with Hash algorithm.
	sourceAddrHashEventLoopSet struct {
		eventLoops []*eventloop
		size       int
	}
)

// ==================================== Implementation of Round-Robin load-balancer ====================================

func (set *roundRobinEventLoopSet) register(el *eventloop) {
	el.idx = set.size
	set.eventLoops = append(set.eventLoops, el)
	set.size++
}

// next returns the eligible event-loop based on Round-Robin algorithm.
func (set *roundRobinEventLoopSet) next(_ int) (el *eventloop) {
	el = set.eventLoops[set.nextLoopIndex]
	if set.nextLoopIndex++; set.nextLoopIndex >= set.size {
		set.nextLoopIndex = 0
	}
	return
}

func (set *roundRobinEventLoopSet) iterate(f func(int, *eventloop) bool) {
	for i, el := range set.eventLoops {
		if !f(i, el) {
			break
		}
	}
}

func (set *roundRobinEventLoopSet) len() int {
	return set.size
}

func (set *roundRobinEventLoopSet) calibrate(el *eventloop, delta int32) {
	atomic.AddInt32(&el.connCount, delta)
}

// ================================= Implementation of Least-Connections load-balancer =================================

// Leverage min-heap to optimize Least-Connections load-balancing.
type minEventLoopHeap []*eventloop

// Implement heap.Interface: Len, Less, Swap, Push, Pop.
func (h minEventLoopHeap) Len() int {
	return len(h)
}

func (h minEventLoopHeap) Less(i, j int) bool {
	// return (*h)[i].loadConnCount() < (*h)[j].loadConnCount()
	return h[i].connCount < h[j].connCount
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

func (set *leastConnectionsEventLoopSet) register(el *eventloop) {
	set.Lock()
	heap.Push(&set.minHeap, el)
	if el.idx == 0 {
		set.cachedRoot = el
	}
	set.calibrateConnsThreshold = int32(set.minHeap.Len())
	set.Unlock()
}

// next returns the eligible event-loop by taking the root node from minimum heap based on Least-Connections algorithm.
func (set *leastConnectionsEventLoopSet) next(_ int) (el *eventloop) {
	// set.RLock()
	// el = set.minHeap[0]
	// set.RUnlock()
	// return

	// In most cases, `next` method returns the cached event-loop immediately and it only reconstructs the minimum heap
	// every `calibrateConnsThreshold` times for reducing locks to global mutex.
	if atomic.LoadInt32(&set.threshold) >= set.calibrateConnsThreshold {
		set.Lock()
		heap.Init(&set.minHeap)
		set.cachedRoot = set.minHeap[0]
		atomic.StoreInt32(&set.threshold, 0)
		set.Unlock()
	}
	return set.cachedRoot
}

func (set *leastConnectionsEventLoopSet) iterate(f func(int, *eventloop) bool) {
	set.RLock()
	for i, el := range set.minHeap {
		if !f(i, el) {
			break
		}
	}
	set.RUnlock()
}

func (set *leastConnectionsEventLoopSet) len() (size int) {
	set.RLock()
	size = set.minHeap.Len()
	set.RUnlock()
	return
}

func (set *leastConnectionsEventLoopSet) calibrate(el *eventloop, delta int32) {
	// set.Lock()
	// el.connCount += delta
	// heap.Fix(&set.minHeap, el.idx)
	// set.Unlock()
	set.RLock()
	atomic.AddInt32(&el.connCount, delta)
	atomic.AddInt32(&set.threshold, 1)
	set.RUnlock()
}

// ======================================= Implementation of Hash load-balancer ========================================

func (set *sourceAddrHashEventLoopSet) register(el *eventloop) {
	el.idx = set.size
	set.eventLoops = append(set.eventLoops, el)
	set.size++
}

// next returns the eligible event-loop by taking the remainder of a given fd as the index of event-loop list.
func (set *sourceAddrHashEventLoopSet) next(hashCode int) *eventloop {
	return set.eventLoops[hashCode%set.size]
}

func (set *sourceAddrHashEventLoopSet) iterate(f func(int, *eventloop) bool) {
	for i, el := range set.eventLoops {
		if !f(i, el) {
			break
		}
	}
}

func (set *sourceAddrHashEventLoopSet) len() int {
	return set.size
}

func (set *sourceAddrHashEventLoopSet) calibrate(el *eventloop, delta int32) {
	atomic.AddInt32(&el.connCount, delta)
}
