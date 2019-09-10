// Copyright 2017 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
)

func main() {
	var port int
	flag.IntVar(&port, "port", 5000, "server port")
	flag.Parse()
	ln, err := net.Listen("tcp4", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	log.Printf("echo server started on port %d", port)
	var id int
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}
		id++
		go func(id int, conn net.Conn) {
			defer func() {
				//log.Printf("closed: %d", id)
				conn.Close()
			}()
			//log.Printf("opened: %d: %s", id, conn.RemoteAddr().String())
			var packet [0xFFF]byte
			for {
				n, err := conn.Read(packet[:])
				if err != nil {
					return
				}
				conn.Write(packet[:n])
			}
		}(id, conn)
	}
}
