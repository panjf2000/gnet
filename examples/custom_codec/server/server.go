package main

import (
	"flag"
	"fmt"
	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/examples/custom_codec/protocol"
	"github.com/panjf2000/gnet/pool/goroutine"
	"log"
	"time"
)

type customCodecServer struct {
	*gnet.EventServer
	addr       string
	multicore  bool
	async      bool
	codec      gnet.ICodec
	workerPool *goroutine.Pool
}

func (cs *customCodecServer) OnInitComplete(srv gnet.Server) (action gnet.Action) {
	log.Printf("Test codec server is listening on %s (multi-cores: %t, loops: %d)\n",
		srv.Addr.String(), srv.Multicore, srv.NumEventLoop)
	return
}

func (cs *customCodecServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	fmt.Println("frame:", string(frame))
	packFrame, _ := protocol.Pack(protocol.DefaultProtocolVersion, protocol.ActionData, frame)
	if cs.async {
		data := append([]byte{}, packFrame...)
		_ = cs.workerPool.Submit(func() {
			c.AsyncWrite(data)
		})
		return
	}
	out = packFrame
	return
}

func testCustomCodecServe(addr string, multicore, async bool, codec gnet.ICodec) {
	var err error
	codec = &protocol.CustomLengthFieldProtocol{}
	cs := &customCodecServer{addr: addr, multicore: multicore, async: async, codec: codec, workerPool: goroutine.Default()}
	err = gnet.Serve(cs, addr, gnet.WithMulticore(multicore), gnet.WithTCPKeepAlive(time.Minute*5), gnet.WithCodec(codec))
	if err != nil {
		panic(err)
	}
}

func main() {
	var port int
	var multicore bool

	// Example command: go run server.go --port 9000 --multicore=true
	flag.IntVar(&port, "port", 9000, "server port")
	flag.BoolVar(&multicore, "multicore", true, "multicore")
	flag.Parse()
	addr := fmt.Sprintf("tcp://:%d", port)
	testCustomCodecServe(addr, multicore, false, nil)
}
