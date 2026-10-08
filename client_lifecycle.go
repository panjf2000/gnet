// Copyright (c) 2026 Telego contributors.
// Licensed under the Apache License, Version 2.0.

package gnet

import (
	"sync"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
)

type clientLifecycle struct {
	mu    sync.Mutex
	done  chan struct{}
	state uint8 // created, starting, running, completed
	err   error
}

// Done closes after the client event loops have exited and closed their sockets.
// On Unix, it also waits for accepted enrollment requests to finish.
// Unexpected loop exits complete automatically, without a Stop call.
func (cli *Client) Done() <-chan struct{} { return cli.lifecycle.done }

// Err returns the terminal client error. Its value is final after Done closes.
func (cli *Client) Err() error {
	cli.lifecycle.mu.Lock()
	defer cli.lifecycle.mu.Unlock()
	return cli.lifecycle.err
}

// Start starts the client once. A stopped or already-started client cannot be
// restarted; concurrent Stop waits until startup and cleanup have completed.
func (cli *Client) Start() error {
	cli.lifecycle.mu.Lock()
	if cli.lifecycle.state != 0 {
		cli.lifecycle.mu.Unlock()
		return errorx.ErrEngineInShutdown
	}
	cli.lifecycle.state = 1
	cli.lifecycle.mu.Unlock()
	err := cli.start()
	cli.lifecycle.mu.Lock()
	cli.lifecycle.state = 2
	cli.lifecycle.mu.Unlock()
	if err != nil {
		cli.eng.shutdown(err)
	}
	go cli.supervise(err)
	if err != nil {
		<-cli.Done()
	}
	return err
}

func (cli *Client) supervise(startErr error) {
	<-cli.eng.concurrency.ctx.Done()
	// A worker can cancel the group before an event-loop reports its exit.
	// Close admission before invoking shutdown callbacks or joining workers.
	cli.eng.shutdown(nil)
	err := cli.waitAndClose()
	if startErr != nil {
		err = startErr
	}
	cli.lifecycle.mu.Lock()
	cli.lifecycle.err = err
	cli.lifecycle.state = 3
	close(cli.lifecycle.done)
	cli.lifecycle.mu.Unlock()
}

// Stop signals shutdown and waits for the same terminal result as Done/Err.
// It is safe to call concurrently and repeatedly, including before Start.
// Like other blocking lifecycle methods, do not call it from an owner callback.
func (cli *Client) Stop() error {
	cli.lifecycle.mu.Lock()
	if cli.lifecycle.state == 0 {
		cli.lifecycle.state = 3
		cli.eng.shutdown(nil)
		cli.eng.inShutdown.Store(true)
		close(cli.lifecycle.done)
		cli.lifecycle.mu.Unlock()
		return nil
	}
	cli.lifecycle.mu.Unlock()
	cli.eng.shutdown(nil)
	<-cli.Done()
	return cli.Err()
}

func (cli *Client) running() bool {
	cli.lifecycle.mu.Lock()
	defer cli.lifecycle.mu.Unlock()
	return cli.lifecycle.state == 2
}
