// Copyright (c) 2023 The Gnet Authors. All rights reserved.
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

//go:build (linux || freebsd || dragonfly || darwin) && gc_opt
// +build linux freebsd dragonfly darwin
// +build gc_opt

package gnet

import (
	"net"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	goPool "github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
)

var testVastConns = false

func TestConnMatrix(t *testing.T) {
	t.Run("1k-connections", func(t *testing.T) {
		testConnMatrix(t, 1000)
	})
	t.Run("10k-connections", func(t *testing.T) {
		testConnMatrix(t, 10000)
	})
	t.Run("100k-connections", func(t *testing.T) {
		testConnMatrix(t, 100000)
	})
	t.Run("1m-connections", func(t *testing.T) {
		if !testVastConns {
			t.Skip("skipped because testVastConns is set to false")
		}
		testConnMatrix(t, 1000000)
	})
}

const (
	actionAdd = iota + 1
	actionDel
)

type handleConn struct {
	c      *conn
	action int
}

func testConnMatrix(t *testing.T, n int) {
	handleConns := make(chan *handleConn, 1024)
	connections := connMatrix{}
	connections.init()
	el := eventloop{engine: &engine{opts: &Options{}}}

	go func() {
		for v := range handleConns {
			switch v.action {
			case actionAdd:
				connections.addConn(v.c, 0)
			case actionDel:
				connections.delConn(v.c)
			default:
				return
			}
		}
	}()

	pool := goPool.Default()
	var m int
	for i := 0; i < n; i++ {
		c := newTCPConn(i, &el, &unix.SockaddrInet4{}, &net.TCPAddr{}, &net.TCPAddr{})
		handleConns <- &handleConn{c, actionAdd}
		if i%2 == 0 {
			m++
			pool.Submit(func() {
				handleConns <- &handleConn{c, actionDel}
			})
		}
	}

	m = n - m

	endTime := time.Now().Add(5 * time.Second)

	for time.Now().Before(endTime) {
		if connections.loadCount() != int32(n)/2 {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		for i := 0; i < len(connections.table); i++ {
			if connections.connCounts[i] == 0 {
				continue
			}

			for j := 0; j < len(connections.table[i]) && m > 0; j++ {
				m--
				c := connections.table[i][j]
				if c == nil {
					t.Fatalf("unexpected nil connection at row %d, column %d", i, j)
				}
				if c.fd != c.gfd.Fd() {
					t.Fatalf("unexpected fd %d, expected fd %d", c.gfd.Fd(), c.fd)
				}
				if i != c.gfd.ConnMatrixRow() || j != c.gfd.ConnMatrixColumn() {
					t.Fatalf("unexpected row %d, column %d, expected row %d, column %d",
						c.gfd.ConnMatrixRow(), c.gfd.ConnMatrixColumn(), i, j)
				}
				gfd, ok := connections.fd2gfd[c.fd]
				if !ok {
					t.Fatalf("missing gfd for fd %d", c.fd)
				}
				if i != gfd.ConnMatrixRow() || j != gfd.ConnMatrixColumn() {
					t.Fatalf("unexpected row %d, column %d, expected row %d, column %d",
						gfd.ConnMatrixRow(), gfd.ConnMatrixColumn(), i, j)
				}
			}
		}

		handleConns <- &handleConn{}

		t.Log("connMatrix remains compact after many additions and deletions, test done!")
		return
	}

	t.Fatalf("test timeout!")
}
