// Copyright 2019 Andy Pan. All rights reserved.
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
)

func main() {
	var port int
	var multicore bool
	var udp bool
	var trace bool
	var reuseport bool

	flag.IntVar(&port, "port", 5000, "server port")
	flag.BoolVar(&udp, "udp", false, "listen on udp")
	flag.BoolVar(&reuseport, "reuseport", false, "reuseport (SO_REUSEPORT)")
	flag.BoolVar(&trace, "trace", false, "print packets to console")
	flag.BoolVar(&multicore, "multicore", true, "multicore")
	flag.Parse()

	var events gnet.Events
	events.Multicore = multicore
	events.OnInitComplete = func(srv gnet.Server) (action gnet.Action) {
		log.Printf("echo server started on port %d (loops: %d)", port, srv.NumLoops)
		if reuseport {
			log.Printf("reuseport")
		}
		return
	}
	events.React = func(c gnet.Conn) (action gnet.Action) {
		top, tail := c.Read()
		c.Write(append(top, tail...))
		c.ResetBuffer()
		if trace {
			log.Printf("%s", strings.TrimSpace(string(top)+string(tail)))
		}
		return
	}
	scheme := "tcp"
	if udp {
		scheme = "udp"
	}
	log.Fatal(gnet.Serve(events, fmt.Sprintf("%s://:%d", scheme, port)))
}
