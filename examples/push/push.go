package main

import (
	"flag"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/gnet"
)

type pushServer struct {
	*gnet.EventServer
	tick             time.Duration
	total            int32
	connectedSockets sync.Map
}

func (ps *pushServer) OnInitComplete(srv gnet.Server) (action gnet.Action) {
	log.Printf("Push server is listening on %s (multi-cores: %t, loops: %d), "+
		"pushing data every %s ...\n", srv.Addr.String(), srv.Multicore, srv.NumEventLoop, ps.tick.String())
	return
}
func (ps *pushServer) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	log.Printf("Socket with addr: %s has been opened...\n", c.RemoteAddr().String())
	atomic.AddInt32(&ps.total, 1)
	ps.connectedSockets.Store(c.RemoteAddr().String(), c)
	return
}
func (ps *pushServer) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	//log.Printf("Socket with addr: %s is closing...\n", c.RemoteAddr().String())
	atomic.AddInt32(&ps.total, -1)
	ps.connectedSockets.Delete(c.RemoteAddr().String())
	return
}
func (ps *pushServer) Tick() (delay time.Duration, action gnet.Action) {
	log.Println("已链接 - %d", atomic.LoadInt32(&ps.total))
	return 5 * time.Second, gnet.None
}
func (ps *pushServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	out = frame
	return
}

func main() {
	var port int
	var multicore bool
	var interval time.Duration
	var ticker bool

	// Example command: go run push.go --port 9000 --tick 1s --multicore=true
	flag.IntVar(&port, "port", 9000, "server port")
	flag.BoolVar(&multicore, "multicore", true, "multicore")
	flag.DurationVar(&interval, "tick", 0, "pushing tick")
	flag.Parse()
	if interval > 0 {
		ticker = true
	}
	push := &pushServer{tick: interval}
	log.Fatal(gnet.Serve(push, fmt.Sprintf("tcp://:%d", port), gnet.WithMulticore(multicore), gnet.WithTicker(ticker), gnet.WithLoadBalance(gnet.Priority)))
}
