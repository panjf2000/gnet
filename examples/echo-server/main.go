// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2017 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/panjf2000/gnet"
)

func main() {
	var port int
	var multicore bool

	flag.IntVar(&port, "port", 5000, "server port")
	flag.BoolVar(&multicore, "multicore", true, "multicore")
	flag.Parse()

	var events gnet.Events
	events.OnInitComplete = func(srv gnet.Server) (action gnet.Action) {
		log.Printf("Server is running under multi-core: %t, loops: %d\n", srv.Multicore, srv.NumLoops)
		return
	}
	events.React = func(c gnet.Conn) (out []byte, action gnet.Action) {
		top, tail := c.ReadPair()
		out = top
		if tail != nil {
			out = append(top, tail...)
		}
		c.ResetBuffer()
		return
	}
	scheme := "tcp"
	log.Fatal(gnet.Serve(events, fmt.Sprintf("%s://:%d", scheme, port), gnet.WithMulticore(multicore)))
}
