package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/panjf2000/gnet/examples/custom_codec/protocol"
	"io"
	"log"
	"net"
)

// Example command: go run client.go
func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	go func() {
		for {

			response, err := ClientDecode(conn)

			if err != nil {
				log.Printf("ClientDecode error, %v\n", err)
			}

			log.Printf("receive , %v, data:%s\n", response, string(response.Data))

		}
	}()

	data := []byte("hello")
	pbdata, err := protocol.Pack(protocol.PROTOCAL_VERSION, protocol.ACTION_DATA, data)
	if err != nil {
		panic(err)
	}
	//log.Println("pbdata：",pbdata)
	conn.Write(pbdata)

	data = []byte("world")
	pbdata, err = protocol.Pack(protocol.PROTOCAL_VERSION, protocol.ACTION_DATA, data)
	if err != nil {
		panic(err)
	}
	conn.Write(pbdata)

	select {}

}

// ClientDecode :
func ClientDecode(rawConn net.Conn) (*protocol.CustomLengthFieldProtocol, error) {

	newPackage := protocol.CustomLengthFieldProtocol{}

	headData := make([]byte, protocol.DefaultHeadLength)
	n, err := io.ReadFull(rawConn, headData)
	if n != protocol.DefaultHeadLength {
		return nil, err
	}

	//parse protocol header
	bytesBuffer := bytes.NewBuffer(headData)
	binary.Read(bytesBuffer, binary.BigEndian, &newPackage.Version)
	binary.Read(bytesBuffer, binary.BigEndian, &newPackage.ActionType)
	binary.Read(bytesBuffer, binary.BigEndian, &newPackage.DataLength)

	if newPackage.DataLength < 1 {
		return &newPackage, nil
	}

	data := make([]byte, newPackage.DataLength)
	dataNum, err2 := io.ReadFull(rawConn, data)
	if uint32(dataNum) != newPackage.DataLength {
		return nil, errors.New(fmt.Sprintf("read data error, %v", err2))
	}

	newPackage.Data = data

	return &newPackage, nil
}
