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
)

type Client struct {
	opts *Options
	el   *eventloop
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
	eng := &engine{
		listeners:    []*listener{},
		opts:         options,
		turnOff:      shutdown,
		eventHandler: eh,
		concurrency: struct {
			*errgroup.Group
			ctx context.Context
		}{eg, ctx},
	}
	cli.el = &eventloop{
		ch:           make(chan any, 1024),
		eng:          eng,
		connections:  make(map[*conn]struct{}),
		eventHandler: eh,
	}
	return
}

func (cli *Client) Start() error {
	cli.el.eventHandler.OnBoot(Engine{cli.el.eng})
	cli.el.eng.concurrency.Go(cli.el.run)
	if cli.opts.Ticker {
		ctx := cli.el.eng.concurrency.ctx
		cli.el.eng.concurrency.Go(func() error {
			cli.el.ticker(ctx)
			return nil
		})
	}
	logging.Debugf("default logging level is %s", logging.LogLevel())
	return nil
}

func (cli *Client) Stop() (err error) {
	cli.el.ch <- errorx.ErrEngineShutdown
	err = cli.el.eng.concurrency.Wait()
	cli.el.eventHandler.OnShutdown(Engine{cli.el.eng})
	logging.Cleanup()
	return
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
	connOpened := make(chan struct{})
	switch v := nc.(type) {
	case *net.TCPConn:
		if cli.opts.TCPNoDelay == TCPNoDelay {
			if err = v.SetNoDelay(true); err != nil {
				return
			}
		}
		if cli.opts.TCPKeepAlive > 0 {
			if err = v.SetKeepAlive(true); err != nil {
				return
			}
			if err = v.SetKeepAlivePeriod(cli.opts.TCPKeepAlive); err != nil {
				return
			}
		}

		c := newTCPConn(nc, cli.el)
		c.SetContext(ctx)
		cli.el.ch <- &openConn{c: c, cb: func() { close(connOpened) }}
		go func(c *conn, tc net.Conn, el *eventloop) {
			var buffer [0x10000]byte
			for {
				n, err := tc.Read(buffer[:])
				if err != nil {
					el.ch <- &netErr{c, err}
					return
				}
				el.ch <- packTCPConn(c, buffer[:n])
			}
		}(c, nc, cli.el)
		gc = c
	case *net.UnixConn:
		c := newTCPConn(nc, cli.el)
		c.SetContext(ctx)
		cli.el.ch <- &openConn{c: c, cb: func() { close(connOpened) }}
		go func(c *conn, uc net.Conn, el *eventloop) {
			var buffer [0x10000]byte
			for {
				n, err := uc.Read(buffer[:])
				if err != nil {
					el.ch <- &netErr{c, err}
					mu.RLock()
					tmpDir := unixAddrDirs[uc.LocalAddr().String()]
					mu.RUnlock()
					if err := os.RemoveAll(tmpDir); err != nil {
						logging.Errorf("failed to remove temporary directory for unix local address: %v", err)
					}
					return
				}
				el.ch <- packTCPConn(c, buffer[:n])
			}
		}(c, nc, cli.el)
		gc = c
	case *net.UDPConn:
		c := newUDPConn(cli.el, nil, nc.LocalAddr(), nc.RemoteAddr())
		c.SetContext(ctx)
		c.rawConn = nc
		cli.el.ch <- &openConn{c: c, cb: func() { close(connOpened) }}
		go func(uc net.Conn, el *eventloop) {
			var buffer [0x10000]byte
			for {
				n, err := uc.Read(buffer[:])
				if err != nil {
					return
				}
				c := newUDPConn(cli.el, nil, uc.LocalAddr(), uc.RemoteAddr())
				c.SetContext(ctx)
				c.rawConn = uc
				el.ch <- packUDPConn(c, buffer[:n])
			}
		}(nc, cli.el)
		gc = c
	default:
		return nil, errorx.ErrUnsupportedProtocol
	}

	<-connOpened
	return
}
