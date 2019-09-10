// Copyright 2017 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/panjf2000/gnet"
	"github.com/tidwall/redcon"
)

type conn struct {
	is   evio.InputStream
	addr string
}

func main() {
	var port int
	var unixsocket string
	var stdlib bool
	var loops int
	var balance string
	flag.IntVar(&port, "port", 6380, "server port")
	flag.IntVar(&loops, "loops", 0, "num loops")
	flag.StringVar(&unixsocket, "unixsocket", "socket", "unix socket")
	flag.StringVar(&balance, "balance", "random", "random, round-robin, least-connections")
	flag.BoolVar(&stdlib, "stdlib", false, "use stdlib")
	flag.Parse()

	var mu sync.RWMutex
	var keys = make(map[string]string)
	var events evio.Events
	switch balance {
	default:
		log.Fatalf("invalid -balance flag: '%v'", balance)
	case "random":
		events.LoadBalance = evio.Random
	case "round-robin":
		events.LoadBalance = evio.RoundRobin
	case "least-connections":
		events.LoadBalance = evio.LeastConnections
	}
	events.NumLoops = loops
	events.Serving = func(srv evio.Server) (action evio.Action) {
		log.Printf("redis server started on port %d (loops: %d)", port, srv.NumLoops)
		if unixsocket != "" {
			log.Printf("redis server started at %s (loops: %d)", unixsocket, srv.NumLoops)
		}
		if stdlib {
			log.Printf("stdlib")
		}
		return
	}
	events.Opened = func(ec evio.Conn) (out []byte, opts evio.Options, action evio.Action) {
		//fmt.Printf("opened: %v\n", ec.RemoteAddr())
		ec.SetContext(&conn{})
		return
	}
	events.Closed = func(ec evio.Conn, err error) (action evio.Action) {
		// fmt.Printf("closed: %v\n", ec.RemoteAddr())
		return
	}

	events.Data = func(ec evio.Conn, in []byte) (out []byte, action evio.Action) {
		if in == nil {
			log.Printf("wake from %s\n", ec.RemoteAddr())
			return nil, evio.Close
		}
		c := ec.Context().(*conn)
		data := c.is.Begin(in)
		var n int
		var complete bool
		var err error
		var args [][]byte
		for action == evio.None {
			complete, args, _, data, err = redcon.ReadNextCommand(data, args[:0])
			if err != nil {
				action = evio.Close
				out = redcon.AppendError(out, err.Error())
				break
			}
			if !complete {
				break
			}
			if len(args) > 0 {
				n++
				switch strings.ToUpper(string(args[0])) {
				default:
					out = redcon.AppendError(out, "ERR unknown command '"+string(args[0])+"'")
				case "PING":
					if len(args) > 2 {
						out = redcon.AppendError(out, "ERR wrong number of arguments for '"+string(args[0])+"' command")
					} else if len(args) == 2 {
						out = redcon.AppendBulk(out, args[1])
					} else {
						out = redcon.AppendString(out, "PONG")
					}
				case "WAKE":
					go ec.Wake()
					out = redcon.AppendString(out, "OK")
				case "ECHO":
					if len(args) != 2 {
						out = redcon.AppendError(out, "ERR wrong number of arguments for '"+string(args[0])+"' command")
					} else {
						out = redcon.AppendBulk(out, args[1])
					}
				case "SHUTDOWN":
					out = redcon.AppendString(out, "OK")
					action = evio.Shutdown
				case "QUIT":
					out = redcon.AppendString(out, "OK")
					action = evio.Close
				case "GET":
					if len(args) != 2 {
						out = redcon.AppendError(out, "ERR wrong number of arguments for '"+string(args[0])+"' command")
					} else {
						key := string(args[1])
						mu.Lock()
						val, ok := keys[key]
						mu.Unlock()
						if !ok {
							out = redcon.AppendNull(out)
						} else {
							out = redcon.AppendBulkString(out, val)
						}
					}
				case "SET":
					if len(args) != 3 {
						out = redcon.AppendError(out, "ERR wrong number of arguments for '"+string(args[0])+"' command")
					} else {
						key, val := string(args[1]), string(args[2])
						mu.Lock()
						keys[key] = val
						mu.Unlock()
						out = redcon.AppendString(out, "OK")
					}
				case "DEL":
					if len(args) < 2 {
						out = redcon.AppendError(out, "ERR wrong number of arguments for '"+string(args[0])+"' command")
					} else {
						var n int
						mu.Lock()
						for i := 1; i < len(args); i++ {
							if _, ok := keys[string(args[i])]; ok {
								n++
								delete(keys, string(args[i]))
							}
						}
						mu.Unlock()
						out = redcon.AppendInt(out, int64(n))
					}
				case "FLUSHDB":
					mu.Lock()
					keys = make(map[string]string)
					mu.Unlock()
					out = redcon.AppendString(out, "OK")
				}
			}
		}
		c.is.End(data)
		return
	}
	var ssuf string
	if stdlib {
		ssuf = "-net"
	}
	addrs := []string{fmt.Sprintf("tcp"+ssuf+"://:%d", port)}
	if unixsocket != "" {
		addrs = append(addrs, fmt.Sprintf("unix"+ssuf+"://%s", unixsocket))
	}
	err := evio.Serve(events, addrs...)
	if err != nil {
		log.Fatal(err)
	}
}
