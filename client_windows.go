// Copyright (c) 2023 The Gnet Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gnet

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/sync/errgroup"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
	"github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
)

type Client struct {
	opts *Options
	eng  *engine
}

func NewClient(eh EventHandler, opts ...Option) (cli *Client, err error) {
	options := loadOptions(opts...)
	cli = &Client{opts: options}

	logger, logFlusher := logging.GetDefaultLogger(), logging.GetDefaultFlusher()
	if options.Logger == nil {
		if options.LogPath != "" {
			logger, logFlusher, _ = logging.CreateLoggerAsLocalFile(options.LogPath, options.LogLevel)
		}
		options.Logger = logger
	} else {
		logger = options.Logger
		logFlusher = nil
	}
	logging.SetDefaultLoggerAndFlusher(logger, logFlusher)

	rootCtx, shutdown := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(rootCtx)
	eng := engine{
		listeners:    []*listener{},
		opts:         options,
		turnOff:      shutdown,
		eventHandler: eh,
		eventLoops:   new(leastConnectionsLoadBalancer),
		concurrency: struct {
			*errgroup.Group
			ctx context.Context
		}{eg, ctx},
	}
	cli.eng = &eng
	return
}

func (cli *Client) Start() error {
	numEventLoop := determineEventLoops(cli.opts)
	logging.Infof("Starting gnet client with %d event loops", numEventLoop)

	cli.eng.eventHandler.OnBoot(Engine{cli.eng})

	var el0 *eventloop
	for i := 0; i < numEventLoop; i++ {
		el := eventloop{
			ch:           make(chan any, 1024),
			eng:          cli.eng,
			connections:  make(map[*conn]struct{}),
			eventHandler: cli.eng.eventHandler,
		}
		cli.eng.eventLoops.register(&el)
		cli.eng.concurrency.Go(el.run)
		if cli.opts.Ticker && el.idx == 0 {
			el0 = &el
		}
	}

	if el0 != nil {
		ctx := cli.eng.concurrency.ctx
		cli.eng.concurrency.Go(func() error {
			el0.ticker(ctx)
			return nil
		})
	}

	logging.Debugf("default logging level is %s", logging.LogLevel())

	return nil
}

func (cli *Client) Stop() error {
	cli.eng.shutdown(nil)

	cli.eng.eventHandler.OnShutdown(Engine{cli.eng})

	// Notify all event-loops to exit.
	cli.eng.closeEventLoops()

	// Wait for all event-loops to exit.
	err := cli.eng.concurrency.Wait()

	// Put the engine into the shutdown state.
	cli.eng.inShutdown.Store(true)

	// Flush the logger.
	logging.Cleanup()

	return err
}

var (
	mu           sync.RWMutex
	unixAddrDirs = make(map[string]string)
)

// unixAddr uses os.MkdirTemp to get a name that is unique.
func unixAddr(addr string) string {
	// Pass an empty pattern to get a directory name that is as short as possible.
	// If we end up with a name longer than the sun_path field in the sockaddr_un
	// struct, we won't be able to make the syscall to open the socket.
	d, err := os.MkdirTemp("", "")
	if err != nil {
		panic(err)
	}

	tmpAddr := filepath.Join(d, addr)
	mu.Lock()
	unixAddrDirs[tmpAddr] = d
	mu.Unlock()

	return tmpAddr
}

func (cli *Client) Dial(network, addr string) (Conn, error) {
	return cli.DialContext(network, addr, nil)
}

func (cli *Client) DialContext(network, addr string, ctx any) (Conn, error) {
	var (
		c   net.Conn
		err error
	)
	if network == "unix" {
		laddr, _ := net.ResolveUnixAddr(network, unixAddr(addr))
		raddr, _ := net.ResolveUnixAddr(network, addr)
		c, err = net.DialUnix(network, laddr, raddr)
		if err != nil {
			return nil, err
		}
	} else {
		c, err = net.Dial(network, addr)
		if err != nil {
			return nil, err
		}
	}
	return cli.EnrollContext(c, ctx)
}

func (cli *Client) Enroll(nc net.Conn) (gc Conn, err error) {
	return cli.EnrollContext(nc, nil)
}

func (cli *Client) EnrollContext(nc net.Conn, ctx any) (gc Conn, err error) {
	el := cli.eng.eventLoops.next(nil)
	connOpened := make(chan struct{})
	switch v := nc.(type) {
	case *net.TCPConn:
		if cli.opts.TCPNoDelay == TCPNoDelay {
			if err = v.SetNoDelay(true); err != nil {
				return
			}
		}
		c := newStreamConn(el, nc, ctx)
		if opts := cli.opts; opts.TCPKeepAlive > 0 {
			idle := opts.TCPKeepAlive
			intvl := opts.TCPKeepInterval
			if intvl == 0 {
				intvl = opts.TCPKeepAlive / 5
			}
			cnt := opts.TCPKeepCount
			if opts.TCPKeepCount == 0 {
				cnt = 5
			}
			if err = c.SetKeepAlive(true, idle, intvl, cnt); err != nil {
				return
			}
		}
		el.ch <- &openConn{c: c, cb: func() { close(connOpened) }}
		goroutine.DefaultWorkerPool.Submit(func() {
			var buffer [0x10000]byte
			for {
				n, err := nc.Read(buffer[:])
				if err != nil {
					el.ch <- &netErr{c, err}
					return
				}
				el.ch <- packTCPConn(c, buffer[:n])
			}
		})
		gc = c
	case *net.UnixConn:
		c := newStreamConn(el, nc, ctx)
		el.ch <- &openConn{c: c, cb: func() { close(connOpened) }}
		goroutine.DefaultWorkerPool.Submit(func() {
			var buffer [0x10000]byte
			for {
				n, err := nc.Read(buffer[:])
				if err != nil {
					el.ch <- &netErr{c, err}
					mu.RLock()
					tmpDir := unixAddrDirs[nc.LocalAddr().String()]
					mu.RUnlock()
					if err := os.RemoveAll(tmpDir); err != nil {
						logging.Errorf("failed to remove temporary directory for unix local address: %v", err)
					}
					return
				}
				el.ch <- packTCPConn(c, buffer[:n])
			}
		})
		gc = c
	case *net.UDPConn:
		c := newUDPConn(el, nil, nc, nc.LocalAddr(), nc.RemoteAddr(), ctx)
		el.ch <- &openConn{c: c, cb: func() { close(connOpened) }}
		goroutine.DefaultWorkerPool.Submit(func() {
			var buffer [0x10000]byte
			for {
				n, err := nc.Read(buffer[:])
				if err != nil {
					el.ch <- &netErr{c, err}
					return
				}
				c := newUDPConn(el, nil, nc, nc.LocalAddr(), nc.RemoteAddr(), ctx)
				el.ch <- packUDPConn(c, buffer[:n])
			}
		})
		gc = c
	default:
		return nil, errorx.ErrUnsupportedProtocol
	}
	<-connOpened

	return
}
