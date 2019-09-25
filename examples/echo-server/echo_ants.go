package main

import (
	"log"
	"time"

	"github.com/panjf2000/ants"
	"github.com/panjf2000/gnet"
)

func main() {
	var events gnet.Events

	// Create a goroutine pool.
	poolSize := 256 * 1024
	pool, _ := ants.NewPool(poolSize, ants.WithNonblocking(true))
	defer pool.Release()

	events.React = func(c gnet.Conn) (out []byte, action gnet.Action) {
		data := c.ReadBytes()
		c.ResetBuffer()
		// Use ants pool to unblock the event-loop.
		_ = pool.Submit(func() {
			time.Sleep(1 * time.Second)
			c.AsyncWrite(data)
		})
		return
	}
	log.Fatal(gnet.Serve(events, "tcp://:9000", gnet.WithMulticore(true)))
}
