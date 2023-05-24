// Copyright (c) 2021 The Gnet Authors. All rights reserved.
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

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package gnet

import (
	"context"
	"errors"
	"net"
	"strconv"
	"sync"
	"syscall"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/internal/math"
	"github.com/panjf2000/gnet/v2/internal/netpoll"
	"github.com/panjf2000/gnet/v2/internal/socket"
	"github.com/panjf2000/gnet/v2/pkg/buffer/ring"
	gerrors "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

// Client of gnet.
type Client struct {
	opts *Options
	el   *eventloop
}

// NewClient creates an instance of Client.
func NewClient(eh EventHandler, opts ...Option) (cli *Client, err error) {
	options := loadOptions(opts...)
	cli = new(Client)
	cli.opts = options

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

	var p *netpoll.Poller
	if p, err = netpoll.OpenPoller(); err != nil {
		return
	}

	shutdownCtx, shutdown := context.WithCancel(context.Background())
	eng := engine{
		ln:           &listener{},
		opts:         options,
		eventHandler: eh,
		workerPool: struct {
			*errgroup.Group
			shutdownCtx context.Context
			shutdown    context.CancelFunc
			once        sync.Once
		}{&errgroup.Group{}, shutdownCtx, shutdown, sync.Once{}},
	}
	if options.Ticker {
		eng.ticker.ctx, eng.ticker.cancel = context.WithCancel(context.Background())
	}
	el := eventloop{
		ln:     eng.ln,
		engine: &eng,
		poller: p,
	}

	rbc := options.ReadBufferCap
	switch {
	case rbc <= 0:
		options.ReadBufferCap = MaxStreamBufferCap
	case rbc <= ring.DefaultBufferSize:
		options.ReadBufferCap = ring.DefaultBufferSize
	default:
		options.ReadBufferCap = math.CeilToPowerOfTwo(rbc)
	}
	wbc := options.WriteBufferCap
	switch {
	case wbc <= 0:
		options.WriteBufferCap = MaxStreamBufferCap
	case wbc <= ring.DefaultBufferSize:
		options.WriteBufferCap = ring.DefaultBufferSize
	default:
		options.WriteBufferCap = math.CeilToPowerOfTwo(wbc)
	}

	el.buffer = make([]byte, options.ReadBufferCap)
	el.connections.init()
	el.eventHandler = eh
	cli.el = &el
	return
}

// Start starts the client event-loop, handing IO events.
func (cli *Client) Start() error {
	cli.el.eventHandler.OnBoot(Engine{})
	cli.el.engine.workerPool.Go(cli.el.run)
	// Start the ticker.
	if cli.opts.Ticker {
		go cli.el.ticker(cli.el.engine.ticker.ctx)
	}
	logging.Debugf("default logging level is %s", logging.LogLevel())
	return nil
}

// Stop stops the client event-loop.
func (cli *Client) Stop() (err error) {
	logging.Error(cli.el.poller.UrgentTrigger(func(_ interface{}) error { return gerrors.ErrEngineShutdown }, nil))
	// Stop the ticker.
	if cli.opts.Ticker {
		cli.el.engine.ticker.cancel()
	}
	_ = cli.el.engine.workerPool.Wait()
	logging.Error(cli.el.poller.Close())
	cli.el.eventHandler.OnShutdown(Engine{})
	logging.Cleanup()
	return
}

// Dial is like net.Dial().
func (cli *Client) Dial(network, address string) (Conn, error) {
	c, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return cli.Enroll(c)
}

// Enroll converts a net.Conn to gnet.Conn and then adds it into Client.
func (cli *Client) Enroll(c net.Conn) (Conn, error) {
	defer c.Close()

	sc, ok := c.(syscall.Conn)
	if !ok {
		return nil, errors.New("failed to convert net.Conn to syscall.Conn")
	}
	rc, err := sc.SyscallConn()
	if err != nil {
		return nil, errors.New("failed to get syscall.RawConn from net.Conn")
	}

	var dupFD int
	e := rc.Control(func(fd uintptr) {
		dupFD, err = unix.Dup(int(fd))
	})
	if err != nil {
		return nil, err
	}
	if e != nil {
		return nil, e
	}

	if cli.opts.SocketSendBuffer > 0 {
		if err = socket.SetSendBuffer(dupFD, cli.opts.SocketSendBuffer); err != nil {
			return nil, err
		}
	}
	if cli.opts.SocketRecvBuffer > 0 {
		if err = socket.SetRecvBuffer(dupFD, cli.opts.SocketRecvBuffer); err != nil {
			return nil, err
		}
	}

	var (
		sockAddr unix.Sockaddr
		gc       Conn
	)
	switch c.(type) {
	case *net.UnixConn:
		if sockAddr, _, _, err = socket.GetUnixSockAddr(c.RemoteAddr().Network(), c.RemoteAddr().String()); err != nil {
			return nil, err
		}
		ua := c.LocalAddr().(*net.UnixAddr)
		ua.Name = c.RemoteAddr().String() + "." + strconv.Itoa(dupFD)
		gc = newTCPConn(dupFD, cli.el, sockAddr, c.LocalAddr(), c.RemoteAddr())
	case *net.TCPConn:
		if cli.opts.TCPNoDelay == TCPDelay {
			if err = socket.SetNoDelay(dupFD, 0); err != nil {
				return nil, err
			}
		}
		if cli.opts.TCPKeepAlive > 0 {
			if err = socket.SetKeepAlivePeriod(dupFD, int(cli.opts.TCPKeepAlive.Seconds())); err != nil {
				return nil, err
			}
		}
		if sockAddr, _, _, _, err = socket.GetTCPSockAddr(c.RemoteAddr().Network(), c.RemoteAddr().String()); err != nil {
			return nil, err
		}
		gc = newTCPConn(dupFD, cli.el, sockAddr, c.LocalAddr(), c.RemoteAddr())
	case *net.UDPConn:
		if sockAddr, _, _, _, err = socket.GetUDPSockAddr(c.RemoteAddr().Network(), c.RemoteAddr().String()); err != nil {
			return nil, err
		}
		gc = newUDPConn(dupFD, cli.el, c.LocalAddr(), sockAddr, true)
	default:
		return nil, gerrors.ErrUnsupportedProtocol
	}
	err = cli.el.poller.UrgentTrigger(cli.el.register, gc)
	if err != nil {
		gc.Close()
		return nil, err
	}
	return gc, nil
}
