// Copyright (c) 2021 Andy Pan
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

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/internal/netpoll"
	"github.com/panjf2000/gnet/v2/internal/socket"
	"github.com/panjf2000/gnet/v2/internal/toolkit"
	"github.com/panjf2000/gnet/v2/pkg/buffer/ring"
	gerrors "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

// Client of gnet.
type Client struct {
	opts     *Options
	el       *eventloop
	logFlush func() error
}

// NewClient creates an instance of Client.
func NewClient(eventHandler EventHandler, opts ...Option) (cli *Client, err error) {
	options := loadOptions(opts...)
	cli = new(Client)
	cli.opts = options
	var logger logging.Logger
	if options.LogPath != "" {
		if logger, cli.logFlush, err = logging.CreateLoggerAsLocalFile(options.LogPath, options.LogLevel); err != nil {
			return
		}
	} else {
		logger = logging.GetDefaultLogger()
	}
	if options.Logger == nil {
		options.Logger = logger
	}
	var p *netpoll.Poller
	if p, err = netpoll.OpenPoller(); err != nil {
		return
	}
	eng := new(engine)
	eng.opts = options
	eng.eventHandler = eventHandler
	eng.ln = &listener{network: "udp"}
	eng.cond = sync.NewCond(&sync.Mutex{})
	if options.Ticker {
		eng.tickerCtx, eng.cancelTicker = context.WithCancel(context.Background())
	}
	el := new(eventloop)
	el.ln = eng.ln
	el.engine = eng
	el.poller = p

	rbc := options.ReadBufferCap
	switch {
	case rbc <= 0:
		options.ReadBufferCap = MaxStreamBufferCap
	case rbc <= ring.DefaultBufferSize:
		options.ReadBufferCap = ring.DefaultBufferSize
	default:
		options.ReadBufferCap = toolkit.CeilToPowerOfTwo(rbc)
	}
	wbc := options.WriteBufferCap
	switch {
	case wbc <= 0:
		options.WriteBufferCap = MaxStreamBufferCap
	case wbc <= ring.DefaultBufferSize:
		options.WriteBufferCap = ring.DefaultBufferSize
	default:
		options.WriteBufferCap = toolkit.CeilToPowerOfTwo(wbc)
	}

	el.buffer = make([]byte, options.ReadBufferCap)
	el.udpSockets = make(map[int]*conn)
	el.connections = make(map[int]*conn)
	el.eventHandler = eventHandler
	cli.el = el
	return
}

// Start starts the client event-loop, handing IO events.
func (cli *Client) Start() error {
	cli.el.eventHandler.OnBoot(Engine{})
	cli.el.engine.wg.Add(1)
	go func() {
		cli.el.run(cli.opts.LockOSThread)
		cli.el.engine.wg.Done()
	}()
	// Start the ticker.
	if cli.opts.Ticker {
		go cli.el.ticker(cli.el.engine.tickerCtx)
	}
	return nil
}

// Stop stops the client event-loop.
func (cli *Client) Stop() (err error) {
	logging.Error(cli.el.poller.UrgentTrigger(func(_ interface{}) error { return gerrors.ErrEngineShutdown }, nil))
	cli.el.engine.wg.Wait()
	logging.Error(cli.el.poller.Close())
	cli.el.eventHandler.OnShutdown(Engine{})
	// Stop the ticker.
	if cli.opts.Ticker {
		cli.el.engine.cancelTicker()
	}
	if cli.logFlush != nil {
		err = cli.logFlush()
	}
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
