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

//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package gnet

import (
	"context"
	"net"

	"golang.org/x/sync/errgroup"

	"github.com/panjf2000/gnet/v2/pkg/buffer/ring"
	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
	"github.com/panjf2000/gnet/v2/pkg/math"
	"github.com/panjf2000/gnet/v2/pkg/netpoll"
	"github.com/panjf2000/gnet/v2/pkg/queue"
)

// Client of gnet.
type Client struct {
	lifecycle clientLifecycle
	opts      *Options
	eng       *engine
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

	rootCtx, shutdown := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(rootCtx)
	eng := engine{
		listeners:    make(map[int]*listener),
		opts:         options,
		turnOff:      shutdown,
		eventHandler: eh,
		eventLoops:   new(leastConnectionsLoadBalancer),
		concurrency: struct {
			*errgroup.Group
			ctx context.Context
		}{eg, ctx},
	}

	if options.EdgeTriggeredIOChunk > 0 {
		options.EdgeTriggeredIO = true
		options.EdgeTriggeredIOChunk = math.CeilToPowerOfTwo(options.EdgeTriggeredIOChunk)
	} else if options.EdgeTriggeredIO {
		options.EdgeTriggeredIOChunk = 1 << 20 // 1MB
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
	cli.lifecycle.done = make(chan struct{})
	cli.eng = &eng
	return
}

// start prepares every loop before the lifecycle supervisor can join them.
func (cli *Client) start() error {
	numEventLoop := determineEventLoops(cli.opts)
	logging.Infof("Starting gnet client with %d event loops", numEventLoop)

	if cli.eng.eventHandler.OnBoot(Engine{cli.eng}) == Shutdown {
		cli.eng.shutdown(nil)
		return nil
	}

	var el0 *eventloop
	for i := 0; i < numEventLoop; i++ {
		p, err := netpoll.OpenPoller()
		if err != nil {
			return err
		}
		el := eventloop{
			listeners:    cli.eng.listeners,
			engine:       cli.eng,
			poller:       p,
			buffer:       make([]byte, cli.opts.ReadBufferCap),
			eventHandler: cli.eng.eventHandler,
		}
		el.connections.init()
		cli.eng.eventLoops.register(&el)
		if cli.opts.Ticker && el.idx == 0 {
			el0 = &el
		}
	}

	cli.eng.eventLoops.iterate(func(_ int, el *eventloop) bool {
		cli.eng.concurrency.Go(el.run)
		return true
	})

	// Start the ticker.
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

// waitAndClose runs once after startup has finished and shutdown was signaled.
func (cli *Client) waitAndClose() error {
	cli.eng.eventHandler.OnShutdown(Engine{cli.eng})

	// Notify all event-loops to exit.
	cli.eng.eventLoops.iterate(func(_ int, el *eventloop) bool {
		logging.Error(el.poller.Trigger(queue.HighPriority,
			func(_ any) error { return errorx.ErrEngineShutdown }, nil))
		return true
	})

	// Wait for all event-loops to exit.
	err := cli.eng.concurrency.Wait()

	cli.eng.enrollments.Wait()
	cli.eng.closeEventLoops()

	// Put the engine into the shutdown state.
	cli.eng.inShutdown.Store(true)

	// Flush the logger.
	logging.Cleanup()

	return err
}

// Dial is like net.Dial().
func (cli *Client) Dial(network, address string) (Conn, error) {
	return cli.DialContext(network, address, nil)
}

// DialContext is like Dial but also accepts an empty interface ctx that can be obtained later via Conn.Context.
func (cli *Client) DialContext(network, address string, ctx any) (Conn, error) {
	if !cli.running() || !cli.eng.beginEnrollment() {
		return nil, errorx.ErrEngineInShutdown
	}
	defer cli.eng.enrollments.Done()
	el := cli.eng.eventLoops.next(nil)
	result := el.enrollConnection(context.Background(), nil, clientDialAddress{network, address}, ctx, queue.HighPriority, true)
	return result.Conn, result.Err
}

type clientDialAddress struct{ network, address string }

func (a clientDialAddress) Network() string { return a.network }
func (a clientDialAddress) String() string  { return a.address }

// Enroll converts a net.Conn to gnet.Conn and then adds it into the Client.
func (cli *Client) Enroll(c net.Conn) (Conn, error) {
	return cli.EnrollContext(c, nil)
}

// EnrollContext is like Enroll but also accepts an empty interface ctx that can be obtained later via Conn.Context.
// It takes ownership of every nonnil c, including when enrollment is rejected.
func (cli *Client) EnrollContext(c net.Conn, ctx any) (Conn, error) {
	if c == nil {
		return nil, errorx.ErrInvalidNetConn
	}
	if !cli.running() || !cli.eng.beginEnrollment() {
		_ = c.Close()
		return nil, errorx.ErrEngineInShutdown
	}
	defer cli.eng.enrollments.Done()
	el := cli.eng.eventLoops.next(nil)
	result := el.enrollConnection(context.Background(), c, c.RemoteAddr(), ctx, queue.HighPriority, true)
	return result.Conn, result.Err
}
