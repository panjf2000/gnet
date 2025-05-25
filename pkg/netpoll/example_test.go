// Copyright (c) 2025 The Gnet Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package netpoll_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/netpoll"
)

func Example() {
	ln, err := net.Listen("tcp", "127.0.0.1:9090")
	if err != nil {
		panic(fmt.Sprintf("Error listening: %v", err))
	}

	defer ln.Close() //nolint:errcheck

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	go func() {
		c, err := ln.Accept()
		if err != nil {
			panic(fmt.Sprintf("Error accepting connection: %v", err))
		}

		defer c.Close() //nolint:errcheck

		buf := make([]byte, 64)

		for {
			select {
			case <-ctx.Done():
				cancel()
				fmt.Printf("Signal received: %v\n", ctx.Err())
				return
			default:
			}

			_, err := c.Read(buf)
			if err != nil {
				panic(fmt.Sprintf("Error reading data from client: %v", err))
			}
			fmt.Printf("Received data from client: %s\n", buf)

			_, err = c.Write([]byte("Hello, client!"))
			if err != nil {
				panic(fmt.Sprintf("Error writing data to client: %v", err))
			}
			fmt.Println("Sent data to client")

			time.Sleep(200 * time.Millisecond)
		}
	}()

	// Wait for the server to start running.
	time.Sleep(500 * time.Millisecond)

	poller, err := netpoll.OpenPoller()
	if err != nil {
		panic(fmt.Sprintf("Error opening poller: %v", err))
	}

	defer poller.Close() //nolint:errcheck

	addr, err := net.ResolveTCPAddr("tcp", ln.Addr().String())
	if err != nil {
		panic(fmt.Sprintf("Error resolving TCP address: %v", err))
	}
	c, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		panic(fmt.Sprintf("Error dialing TCP address: %v", err))
	}

	f, err := c.File()
	if err != nil {
		panic(fmt.Sprintf("Error getting file from connection: %v", err))
	}

	closeClient := func() {
		c.Close() //nolint:errcheck
		f.Close() //nolint:errcheck
	}
	defer closeClient()

	sendData := true

	pa := netpoll.PollAttachment{
		FD: int(f.Fd()),
		Callback: func(fd int, event netpoll.IOEvent, flags netpoll.IOFlags) error { //nolint:revive
			if netpoll.IsErrorEvent(event, flags) {
				closeClient()
				return errors.ErrEngineShutdown
			}

			if netpoll.IsReadEvent(event) {
				sendData = true
				buf := make([]byte, 64)
				_, err := c.Read(buf)
				if err != nil {
					closeClient()
					fmt.Println("Error reading data from server:", err)
					return errors.ErrEngineShutdown
				}
				fmt.Printf("Received data from server: %s\n", buf)
				// Process the data...
			}

			if netpoll.IsWriteEvent(event) && sendData {
				sendData = false
				// Write data to the connection...
				_, err := c.Write([]byte("Hello, server!"))
				if err != nil {
					closeClient()
					fmt.Println("Error writing data to server:", err)
					return errors.ErrEngineShutdown
				}
				fmt.Println("Sent data to server")
			}

			return nil
		},
	}

	if err := poller.AddReadWrite(&pa, false); err != nil {
		panic(fmt.Sprintf("Error adding file descriptor to poller: %v", err))
	}

	err = poller.Polling(func(fd int, event netpoll.IOEvent, flags netpoll.IOFlags) error {
		return pa.Callback(fd, event, flags)
	})

	fmt.Printf("Poller exited with error: %v", err)
}
