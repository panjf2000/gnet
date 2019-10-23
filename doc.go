/*
gnet is an event-driven networking framework that is fast and small. It makes direct epoll and kqueue syscalls rather
than using the standard Go net package, and works in a similar manner as netty and libuv.

The goal of this project is to create a server framework for Go that performs on par with Redis and Haproxy for
packet handling.

gnet sells itself as a high-performance, lightweight, non-blocking, event-driven networking framework written in pure Go
which works on transport layer with TCP/UDP/Unix-Socket protocols, so it allows developers to implement their own
protocols of application layer upon gnet for building diversified network applications, for instance,
you get an HTTP Server or Web Framework if you implement HTTP protocol upon gnet while you have a Redis Server done
with the implementation of Redis protocol upon gnet and so on.

Echo server built upon gnet is shown below:

	package main

	import (
		"log"

		"github.com/panjf2000/gnet"
	)

	type echoServer struct {
		*gnet.EventServer
	}

	func (es *echoServer) React(c gnet.Conn) (out []byte, action gnet.Action) {
		out = c.Read()
		c.ResetBuffer()
		return
	}

	func main() {
		echo := new(echoServer)
		log.Fatal(gnet.Serve(echo, "tcp://:9000", gnet.WithMulticore(true)))
	}
*/
package gnet
