package queue_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/panjf2000/gnet/internal/queue"
)

func TestLockFreeQueue(t *testing.T) {
	const taskNum = 10000
	q := queue.NewLockFreeQueue()
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		for i := 0; i < taskNum; i++ {
			task := &queue.Task{}
			q.Enqueue(task)
		}
		wg.Done()
	}()
	go func() {
		for i := 0; i < taskNum; i++ {
			task := &queue.Task{}
			q.Enqueue(task)
		}
		wg.Done()
	}()

	var counter int32
	go func() {
		for {
			task := q.Dequeue()
			if task != nil {
				atomic.AddInt32(&counter, 1)
			}
			if task == nil && atomic.LoadInt32(&counter) == 2*taskNum {
				break
			}
		}
		wg.Done()
	}()
	go func() {
		for {
			task := q.Dequeue()
			if task != nil {
				atomic.AddInt32(&counter, 1)
			}
			if task == nil && atomic.LoadInt32(&counter) == 2*taskNum {
				break
			}
		}
		wg.Done()
	}()
	wg.Wait()

	t.Logf("sent and received all %d tasks", 2*taskNum)
}
