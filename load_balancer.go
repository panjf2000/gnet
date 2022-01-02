// Copyright (c) 2019 Andy Pan
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gnet

import (
	"hash/crc32"
	"net"

	"github.com/panjf2000/gnet/v2/internal/toolkit"
)

// LoadBalancing represents the type of load-balancing algorithm.
type LoadBalancing int

const (
	// RoundRobin assigns the next accepted connection to the event-loop by polling event-loop list.
	RoundRobin LoadBalancing = iota

	// LeastConnections assigns the next accepted connection to the event-loop that is
	// serving the least number of active connections at the current time.
	LeastConnections

	// SourceAddrHash assigns the next accepted connection to the event-loop by hashing the remote address.
	SourceAddrHash
)

type (
	// loadBalancer is an interface which manipulates the event-loop set.
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
		eventLoops []*eventloop
		size       int
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

func (lb *leastConnectionsLoadBalancer) min() (el *eventloop) {
	el = lb.eventLoops[0]
	minN := el.loadConn()
	for _, v := range lb.eventLoops[1:] {
		if n := v.loadConn(); n < minN {
			minN = n
			el = v
		}
	}
	return
}

func (lb *leastConnectionsLoadBalancer) register(el *eventloop) {
	el.idx = lb.size
	lb.eventLoops = append(lb.eventLoops, el)
	lb.size++
}

// next returns the eligible event-loop by taking the root node from minimum heap based on Least-Connections algorithm.
func (lb *leastConnectionsLoadBalancer) next(_ net.Addr) (el *eventloop) {
	return lb.min()
}

func (lb *leastConnectionsLoadBalancer) iterate(f func(int, *eventloop) bool) {
	for i, el := range lb.eventLoops {
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

// hash converts a string to a unique hash code.
func (lb *sourceAddrHashLoadBalancer) hash(s string) int {
	v := int(crc32.ChecksumIEEE(toolkit.StringToBytes(s)))
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
