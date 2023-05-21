// Copyright (c) 2019 The Gnet Authors. All rights reserved.
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

	"github.com/panjf2000/gnet/v2/internal/bs"
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
		index(int) *eventloop
		iterate(func(int, *eventloop) bool)
		len() int
	}

	// baseLoadBalancer with base lb.
	baseLoadBalancer struct {
		eventLoops []*eventloop
		size       int
	}

	// roundRobinLoadBalancer with Round-Robin algorithm.
	roundRobinLoadBalancer struct {
		baseLoadBalancer
		nextIndex uint64
	}

	// leastConnectionsLoadBalancer with Least-Connections algorithm.
	leastConnectionsLoadBalancer struct {
		baseLoadBalancer
	}

	// sourceAddrHashLoadBalancer with Hash algorithm.
	sourceAddrHashLoadBalancer struct {
		baseLoadBalancer
	}
)

// ==================================== Implementation of base load-balancer ====================================

// register adds a new eventloop into load-balancer.
func (lb *baseLoadBalancer) register(el *eventloop) {
	el.idx = lb.size
	lb.eventLoops = append(lb.eventLoops, el)
	lb.size++
}

// index returns the eligible eventloop by index.
func (lb *baseLoadBalancer) index(i int) *eventloop {
	if i >= lb.size {
		return nil
	}
	return lb.eventLoops[i]
}

// iterate iterates all the eventloops.
func (lb *baseLoadBalancer) iterate(f func(int, *eventloop) bool) {
	for i, el := range lb.eventLoops {
		if !f(i, el) {
			break
		}
	}
}

// len returns the length of event-loop list.
func (lb *baseLoadBalancer) len() int {
	return lb.size
}

// ==================================== Implementation of Round-Robin load-balancer ====================================

// next returns the eligible event-loop based on Round-Robin algorithm.
func (lb *roundRobinLoadBalancer) next(_ net.Addr) (el *eventloop) {
	el = lb.eventLoops[lb.nextIndex%uint64(lb.size)]
	lb.nextIndex++
	return
}

// ================================= Implementation of Least-Connections load-balancer =================================

func (lb *leastConnectionsLoadBalancer) next(_ net.Addr) (el *eventloop) {
	el = lb.eventLoops[0]
	minN := el.countConn()
	for _, v := range lb.eventLoops[1:] {
		if n := v.countConn(); n < minN {
			minN = n
			el = v
		}
	}
	return
}

// ======================================= Implementation of Hash load-balancer ========================================

// hash converts a string to a unique hash code.
func (*sourceAddrHashLoadBalancer) hash(s string) int {
	v := int(crc32.ChecksumIEEE(bs.StringToBytes(s)))
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
