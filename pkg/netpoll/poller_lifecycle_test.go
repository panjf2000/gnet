// Copyright (c) 2026 Telego contributors.
// Licensed under the Apache License, Version 2.0. See LICENSE.

//go:build linux

package netpoll

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/pkg/queue"
)

type blockedEnqueue struct {
	queue.AsyncTaskQueue
	entered chan struct{}
	release chan struct{}
	task    *queue.Task
	once    sync.Once
}

func (tasks *blockedEnqueue) Enqueue(task *queue.Task) {
	tasks.AsyncTaskQueue.Enqueue(task)
	tasks.once.Do(func() {
		tasks.task = task
		close(tasks.entered)
		<-tasks.release
	})
}

func TestPollerCloseWaitsForAcceptedTrigger(t *testing.T) {
	poller, err := OpenPoller()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = poller.Close() }()
	blocked := &blockedEnqueue{AsyncTaskQueue: poller.urgentAsyncTaskQueue, entered: make(chan struct{}), release: make(chan struct{})}
	poller.urgentAsyncTaskQueue = blocked
	var executed atomic.Int64
	triggerDone := make(chan error, 1)
	go func() {
		triggerDone <- poller.Trigger(queue.HighPriority, func(any) error { executed.Add(1); return nil }, &executed)
	}()
	<-blocked.entered
	if poller.lifecycle.mu.TryLock() {
		poller.lifecycle.mu.Unlock()
		close(blocked.release)
		t.Fatal("Trigger did not retain admission ownership through notification")
	}
	closeStarted, closeDone := make(chan struct{}), make(chan error, 1)
	go func() { close(closeStarted); closeDone <- poller.Close() }()
	<-closeStarted
	close(blocked.release)
	if err := <-triggerDone; err != nil {
		t.Fatalf("accepted notification raced descriptor closure: %v", err)
	}
	if err := <-closeDone; err != nil {
		t.Fatal(err)
	}
	if executed.Load() != 0 || blocked.task.Exec != nil || blocked.task.Param != nil {
		t.Fatal("Close executed an abandoned task or retained its callback references")
	}
	if poller.asyncTaskQueue != nil || poller.urgentAsyncTaskQueue != nil {
		t.Fatal("Close retained queue heads")
	}
}

func TestPollerCloseRejectsLateTriggerAndPreservesReusedFDs(t *testing.T) {
	poller, err := OpenPoller()
	if err != nil {
		t.Fatal(err)
	}
	pollFD, wakeFD := poller.fd, pollerWakeFD(poller)
	if err := poller.Close(); err != nil {
		t.Fatal(err)
	}
	var replacements []int
	defer func() {
		for _, descriptor := range replacements {
			_ = unix.Close(descriptor)
		}
	}()
	for i := 0; i < 2; i++ {
		descriptor, err := unix.Eventfd(0, unix.EFD_NONBLOCK|unix.EFD_CLOEXEC)
		if err != nil {
			t.Fatal(err)
		}
		replacements = append(replacements, descriptor)
	}
	reused := (replacements[0] == pollFD && replacements[1] == wakeFD) || (replacements[1] == pollFD && replacements[0] == wakeFD)
	if !reused {
		t.Skip("another runtime operation reused a closed descriptor before this test")
	}
	var workers sync.WaitGroup
	for i := 0; i < 32; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 32; j++ {
				if err := poller.Trigger(queue.LowPriority, func(any) error { panic("late callback") }, nil); !errors.Is(err, os.ErrClosed) {
					t.Errorf("late Trigger = %v, want closed", err)
				}
				if err := poller.Close(); err != nil {
					t.Errorf("repeated Close = %v", err)
				}
			}
		}()
	}
	workers.Wait()
	for _, descriptor := range replacements {
		var buffer [8]byte
		if _, err := unix.Read(descriptor, buffer[:]); !errors.Is(err, unix.EAGAIN) {
			t.Fatalf("late Trigger/Close changed reused descriptor %d: %v", descriptor, err)
		}
	}
	if poller.asyncTaskQueue != nil || poller.urgentAsyncTaskQueue != nil {
		t.Fatal("late Trigger restored a queue after closure")
	}
}

func TestPollerTriggerPreservesPriorityAndDiscardsBothQueues(t *testing.T) {
	poller, err := OpenPoller()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = poller.Close() }()
	poller.highPriorityEventsThreshold = 1
	callback := func(any) error { panic("discard must not execute callbacks") }
	if err := poller.Trigger(queue.LowPriority, callback, "initial"); err != nil {
		t.Fatal(err)
	}
	if err := poller.Trigger(queue.LowPriority, callback, "deferred"); err != nil {
		t.Fatal(err)
	}
	if err := poller.Trigger(queue.HighPriority, callback, "urgent"); err != nil {
		t.Fatal(err)
	}
	first := poller.urgentAsyncTaskQueue.Dequeue()
	second := poller.urgentAsyncTaskQueue.Dequeue()
	deferred := poller.asyncTaskQueue.Dequeue()
	if first == nil || second == nil || deferred == nil || first.Param != "initial" || second.Param != "urgent" || deferred.Param != "deferred" {
		t.Fatal("Trigger changed high/low-priority queue semantics")
	}
	poller.urgentAsyncTaskQueue.Enqueue(first)
	poller.urgentAsyncTaskQueue.Enqueue(second)
	poller.asyncTaskQueue.Enqueue(deferred)
	if err := poller.Close(); err != nil {
		t.Fatal(err)
	}
	for _, task := range []*queue.Task{first, second, deferred} {
		if task.Exec != nil || task.Param != nil {
			t.Fatal("Close retained an abandoned callback or parameter")
		}
	}
}

func TestPollerEventfdFailurePreservesStdin(t *testing.T) {
	if os.Getenv("GNET_NETPOLL_FAILURE_CHILD") != "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestPollerEventfdFailurePreservesStdin$")
		command.Env = append(os.Environ(), "GNET_NETPOLL_FAILURE_CHILD=1")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("eventfd failure subprocess: %v\n%s", err, output)
		}
		return
	}
	var limit unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &limit); err != nil {
		t.Fatal(err)
	}
	limit.Cur = 64
	if err := unix.Setrlimit(unix.RLIMIT_NOFILE, &limit); err != nil {
		t.Fatal(err)
	}
	var occupied []int
	defer func() {
		for _, descriptor := range occupied {
			_ = unix.Close(descriptor)
		}
	}()
	for {
		descriptor, err := unix.Open("/dev/null", unix.O_RDONLY|unix.O_CLOEXEC, 0)
		if errors.Is(err, unix.EMFILE) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		occupied = append(occupied, descriptor)
	}
	if len(occupied) == 0 {
		t.Fatal("cannot reserve a descriptor for epoll creation")
	}
	_ = unix.Close(occupied[len(occupied)-1])
	occupied = occupied[:len(occupied)-1]
	poller, err := OpenPoller()
	if poller != nil || !errors.Is(err, unix.EMFILE) {
		t.Fatalf("eventfd exhaustion = %v, %v", poller, err)
	}
	if _, err := unix.FcntlInt(0, unix.F_GETFD, 0); err != nil {
		t.Fatalf("eventfd startup failure closed stdin: %v", err)
	}
}
