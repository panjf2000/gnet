// Copyright (c) 2023 Andy Pan.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package gnet

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

var (
	SysClose = unix.Close
	NetDial  = net.Dial
)

// NOTE: TestServeMulticast can fail with "write: no buffer space available" on wifi interface.
func TestServeMulticast(t *testing.T) {
	t.Run("IPv4", func(t *testing.T) {
		// 224.0.0.169 is an unassigned address from the Local Network Control Block
		// https://www.iana.org/assignments/multicast-addresses/multicast-addresses.xhtml#multicast-addresses-1
		t.Run("udp-multicast", func(t *testing.T) {
			testMulticast(t, "224.0.0.169:9991", false, false, -1, 10)
		})
		t.Run("udp-multicast-reuseport", func(t *testing.T) {
			testMulticast(t, "224.0.0.169:9991", true, false, -1, 10)
		})
		t.Run("udp-multicast-reuseaddr", func(t *testing.T) {
			testMulticast(t, "224.0.0.169:9991", false, true, -1, 10)
		})
	})
	t.Run("IPv6", func(t *testing.T) {
		iface, err := findLoopbackInterface()
		require.NoError(t, err)
		if iface.Flags&net.FlagMulticast != net.FlagMulticast {
			t.Skip("multicast is not supported on loopback interface")
		}
		// ff02::3 is an unassigned address from Link-Local Scope Multicast Addresses
		// https://www.iana.org/assignments/ipv6-multicast-addresses/ipv6-multicast-addresses.xhtml#link-local
		t.Run("udp-multicast", func(t *testing.T) {
			testMulticast(t, fmt.Sprintf("[ff02::3%%%s]:9991", iface.Name), false, false, iface.Index, 10)
		})
		t.Run("udp-multicast-reuseport", func(t *testing.T) {
			testMulticast(t, fmt.Sprintf("[ff02::3%%%s]:9991", iface.Name), true, false, iface.Index, 10)
		})
		t.Run("udp-multicast-reuseaddr", func(t *testing.T) {
			testMulticast(t, fmt.Sprintf("[ff02::3%%%s]:9991", iface.Name), false, true, iface.Index, 10)
		})
	})
}

func findLoopbackInterface() (*net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback == net.FlagLoopback {
			return &iface, nil
		}
	}
	return nil, errors.New("no loopback interface")
}

func testMulticast(t *testing.T, addr string, reuseport, reuseaddr bool, index, nclients int) {
	ts := &testMcastServer{
		t:        t,
		addr:     addr,
		nclients: nclients,
	}
	options := []Option{
		WithReuseAddr(reuseaddr),
		WithReusePort(reuseport),
		WithSocketRecvBuffer(2 * nclients * 1024), // enough space to receive messages from nclients to eliminate dropped packets
		WithTicker(true),
	}
	if index != -1 {
		options = append(options, WithMulticastInterfaceIndex(index))
	}
	err := Run(ts, "udp://"+addr, options...)
	assert.NoError(t, err)
}

type testMcastServer struct {
	*BuiltinEventEngine
	t        *testing.T
	mcast    sync.Map
	addr     string
	nclients int
	started  int32
	active   int32
}

func (s *testMcastServer) startMcastClient() {
	rand.Seed(time.Now().UnixNano())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, err := net.Dial("udp", s.addr)
	require.NoError(s.t, err)
	defer c.Close()
	ch := make(chan []byte, 10000)
	s.mcast.Store(c.LocalAddr().String(), ch)
	duration := time.Duration((rand.Float64()*2+1)*float64(time.Second)) / 2
	s.t.Logf("test duration: %dms", duration/time.Millisecond)
	start := time.Now()
	for time.Since(start) < duration {
		reqData := make([]byte, 1024)
		_, err = rand.Read(reqData)
		require.NoError(s.t, err)
		_, err = c.Write(reqData)
		require.NoError(s.t, err)
		// Workaround for MacOS "write: no buffer space available" error messages
		// https://developer.apple.com/forums/thread/42334
		time.Sleep(time.Millisecond)
		select {
		case respData := <-ch:
			require.Equalf(s.t, reqData, respData, "response mismatch, length of bytes: %d vs %d", len(reqData), len(respData))
		case <-ctx.Done():
			require.Fail(s.t, "timeout receiving message")
			return
		}
	}
}

func (s *testMcastServer) OnTraffic(c Conn) (action Action) {
	buf, _ := c.Next(-1)
	b := make([]byte, len(buf))
	copy(b, buf)
	ch, ok := s.mcast.Load(c.RemoteAddr().String())
	require.True(s.t, ok)
	ch.(chan []byte) <- b
	return
}

func (s *testMcastServer) OnTick() (delay time.Duration, action Action) {
	if atomic.CompareAndSwapInt32(&s.started, 0, 1) {
		for i := 0; i < s.nclients; i++ {
			atomic.AddInt32(&s.active, 1)
			go func() {
				defer atomic.AddInt32(&s.active, -1)
				s.startMcastClient()
			}()
		}
	}
	if atomic.LoadInt32(&s.active) == 0 {
		action = Shutdown
		return
	}
	delay = time.Second / 5
	return
}

type testMulticastBindServer struct {
	*BuiltinEventEngine
}

func (t *testMulticastBindServer) OnTick() (delay time.Duration, action Action) {
	action = Shutdown
	return
}

func TestMulticastBindIPv4(t *testing.T) {
	ts := &testMulticastBindServer{}
	iface, err := findLoopbackInterface()
	require.NoError(t, err)
	err = Run(ts, "udp://224.0.0.169:9991",
		WithMulticastInterfaceIndex(iface.Index),
		WithTicker(true))
	assert.NoError(t, err)
}

func TestMulticastBindIPv6(t *testing.T) {
	ts := &testMulticastBindServer{}
	iface, err := findLoopbackInterface()
	require.NoError(t, err)
	err = Run(ts, fmt.Sprintf("udp://[ff02::3%%%s]:9991", iface.Name),
		WithMulticastInterfaceIndex(iface.Index),
		WithTicker(true))
	assert.NoError(t, err)
}
