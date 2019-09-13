// Copyright 2017 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/ringbuffer"
)

func main() {
	var port int
	var loops int
	var udp bool
	var trace bool
	var reuseport bool

	flag.IntVar(&port, "port", 5000, "server port")
	flag.BoolVar(&udp, "udp", false, "listen on udp")
	flag.BoolVar(&reuseport, "reuseport", false, "reuseport (SO_REUSEPORT)")
	flag.BoolVar(&trace, "trace", false, "print packets to console")
	flag.IntVar(&loops, "loops", 0, "num loops")
	flag.Parse()

	var events gnet.Events
	events.NumLoops = loops
	events.OnInitComplete = func(srv gnet.Server) (action gnet.Action) {
		log.Printf("echo server started on port %d (loops: %d)", port, srv.NumLoops)
		if reuseport {
			log.Printf("reuseport")
		}
		return
	}
	events.React = func(c gnet.Conn, inBuf *ringbuffer.RingBuffer) (out []byte, action gnet.Action) {
		n := inBuf.Length()
		defer inBuf.Move(n)
		//out = inBuf.Bytes()
		defer ringbuffer.Recycle(out)
		if trace {
			log.Printf("%s", strings.TrimSpace(string(out)))
		}
		return
	}
	scheme := "tcp"
	if udp {
		scheme = "udp"
	}
	log.Fatal(gnet.Serve(events, fmt.Sprintf("%s://:%d?reuseport=%t", scheme, port, reuseport)))
}
