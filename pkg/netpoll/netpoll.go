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

/*
Package netpoll provides a portable event-driven interface for network I/O.

The underlying facility of event notification is OS-specific:
  - epoll on Linux - https://man7.org/linux/man-pages/man7/epoll.7.html
  - kqueue on *BSD/Darwin - https://man.freebsd.org/cgi/man.cgi?kqueue

With the help of the netpoll package, you can easily build your own high-performance
event-driven network applications based on epoll/kqueue.

The Poller represents the event notification facility whose backend is epoll or kqueue.
The OpenPoller function creates a new Poller instance:

	poller, err := netpoll.OpenPoller()
	if err != nil {
		// handle error
	}

	defer poller.Close()

	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:9090")
	if err != nil {
		// handle error
	}
	c, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		// handle error
	}

	f, err := c.File()
	if err != nil {
		// handle error
	}

	closeClient := func() {
		c.Close()
		f.Close()
	}
	defer closeClient()

The PollAttachment consists of a file descriptor and its callback function.
PollAttachment is used to register a file descriptor to Poller.
The callback function is called when an event occurs on the file descriptor:

	pa := netpoll.PollAttachment{
		FD: int(f.Fd()),
		Callback: func(fd int, event netpoll.IOEvent, flags netpoll.IOFlags) error {
			if netpoll.IsErrorEvent(event, flags) {
				closeClient()
				return errors.ErrEngineShutdown
			}

			if netpoll.IsReadEvent(event) {
				buf := make([]byte, 64)
				// Read data from the connection.
				_, err := c.Read(buf)
				if err != nil {
					closeClient()
					return errors.ErrEngineShutdown
				}
				// Process the data...
			}

			if netpoll.IsWriteEvent(event) {
				// Write data to the connection.
				_, err := c.Write([]byte("hello"))
				if err != nil {
					closeClient()
					return errors.ErrEngineShutdown
				}
			}

			return nil
		}}

	if err := poller.AddReadWrite(&pa, false); err != nil {
		// handle error
	}

The Poller.Polling function starts the event loop monitoring file descriptors and
waiting for I/O events to occur:

	poller.Polling(func(fd int, event netpoll.IOEvent, flags netpoll.IOFlags) error {
		return pa.Callback(fd, event, flags)
	})

Or

	poller.Polling()

if you've enabled the build tag `poll_opt`.
*/
package netpoll
