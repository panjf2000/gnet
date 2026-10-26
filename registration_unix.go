// Copyright (c) 2026 The Gnet Authors. All rights reserved.
// Licensed under the Apache License, Version 2.0.

//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package gnet

import (
	"errors"
	"os"
	"sync"

	"golang.org/x/sys/unix"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/queue"
)

// connRegistration owns the duplicated descriptor until register0 takes it.
// publication orders the result of Trigger (including a wakeup error after
// enqueue) before the callback may touch the descriptor. Neither lock is held
// while invoking user callbacks, and completion never sends on a channel.
type connRegistration struct {
	c           *conn
	publication sync.Mutex
	submitErr   error
	once        sync.Once
	done        chan struct{}
	err         error
}

func (r *connRegistration) complete(err error) {
	r.once.Do(func() {
		r.err = err
		close(r.done)
	})
}

// discard is called only before execution, after submission failure, or once
// Done proves that Polling can no longer execute this request. A task left in
// the queue after a wakeup error will observe submitErr and never use c.fd.
func (r *connRegistration) discard(err error) {
	r.once.Do(func() {
		r.err = errors.Join(err, closeConnFD(r.c))
		r.c.release()
		close(r.done)
	})
}

func closeConnFD(c *conn) error {
	// Only unpublished registrations use this helper. Published connections
	// keep fd immutable because UDP AsyncWrite may read it concurrently.
	fd := c.fd
	c.fd = -1
	if fd < 0 {
		return nil
	}
	return os.NewSyscallError("close", unix.Close(fd))
}

func (el *eventloop) registerConn(c *conn, priority queue.EventPriority) error {
	r := &connRegistration{c: c, done: make(chan struct{})}
	r.publication.Lock()
	r.submitErr = el.poller.Trigger(priority, el.register, r)
	r.publication.Unlock()
	if r.submitErr != nil {
		r.discard(r.submitErr)
	}
	return r.awaitResult(el.poller.Done())
}

func (r *connRegistration) awaitResult(pollerDone <-chan struct{}) error {
	if !awaitPollerRegister(r.done, pollerDone) {
		r.discard(errorx.ErrEngineShutdown)
	}
	<-r.done
	return r.err
}

func (el *eventloop) registerRequest(r *connRegistration) error {
	r.publication.Lock()
	err := r.submitErr
	r.publication.Unlock()
	if err != nil {
		r.discard(err)
		return nil // The registration error is delivered to its caller.
	}
	err = el.register0(r.c)
	if err != nil && r.c.fd >= 0 {
		// register0 closes its descriptor when AddRead fails. Errors from
		// OnOpen or its first write leave an attached connection to close here.
		err = errors.Join(err, el.close(r.c, err))
	}
	r.complete(err)
	return err
}
