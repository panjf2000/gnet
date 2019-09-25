package main

import (
	"log"
	"strings"

	"github.com/panjf2000/gnet"
)

func main() {
	var trace bool
	var events gnet.Events
	events.React = func(c gnet.Conn) (out []byte, action gnet.Action) {
		top, tail := c.ReadPair()
		out = append(top, tail...)
		c.ResetBuffer()
		if trace {
			log.Printf("%s", strings.TrimSpace(string(top)+string(tail)))
		}
		return
	}
	log.Fatal(gnet.Serve(events, "tcp://:9000", gnet.WithMulticore(true)))
}
