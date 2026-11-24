package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/utils/pool"
)

const POOL_WORKER_COUNT = 3

func TestNewPool(t *testing.T) {
	p := pool.NewPool(POOL_WORKER_COUNT)
	if p == nil {
		t.Errorf("NewPool() returned nil")
	}
}

func TestJobExecution(t *testing.T) {
	p := pool.NewPool(POOL_WORKER_COUNT)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p.Run(ctx)

	done := make(chan bool, 1)
	p.AddJob(func() {
		done <- true
	})

	select {
	case <-done:
		return
	case <-time.After(5 * time.Second):
		t.Error("Job was not executed")
	}
}

func TestLargeNumberOfJobs(t *testing.T) {
	p := pool.NewPool(POOL_WORKER_COUNT)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p.Run(ctx)

	jobCount := 1000
	var wg sync.WaitGroup
	wg.Add(jobCount)

	for i := 0; i < jobCount; i++ {
		p.AddJob(func() {
			wg.Done()
		})
	}

	// Wait for all jobs to be done
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(5 * time.Second):
		t.Error("Not all jobs were processed in time")
	}
}

func TestContextCancelDuringJobExecution(t *testing.T) {
	p := pool.NewPool(POOL_WORKER_COUNT)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p.Run(ctx)

	// A job that a worker has already dequeued must run to completion even
	// if the context is cancelled meanwhile. Wait until the job has started
	// before cancelling: a job still queued at cancel time may be dropped,
	// since the worker select picks randomly between job and ctx.Done().
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	p.AddJob(func() {
		close(started)
		<-release
		close(done)
	})

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		close(release) // unblock the job if it starts late
		t.Fatal("Job did not start")
	}

	cancel()
	close(release)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("Job did not complete as expected after context cancel")
	}
}

func TestAddJobToClosedPool(t *testing.T) {
	p := pool.NewPool(POOL_WORKER_COUNT)
	ctx, cancel := context.WithCancel(context.Background())
	p.Run(ctx)

	cancel()
	p.IsRunning = false

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic occurred while adding job to closed pool: %v", r)
		}
	}()

	done := make(chan bool)
	go func() {
		p.AddJob(func() {
			done <- true
		})
	}()

	select {
	case <-done:
		t.Error("Job should not be processed after context is done")
	case <-time.After(100 * time.Millisecond):
		return
	}
}

func TestConcurrentJobExecution(t *testing.T) {
	p := pool.NewPool(3)
	ctx, cancel := context.WithCancel(context.Background())
	p.Run(ctx)
	defer cancel()

	jobCount := 100
	channel := make(chan int, jobCount)
	var wg sync.WaitGroup
	wg.Add(jobCount)

	for i := 0; i < jobCount; i++ {
		p.AddJob(func() {
			channel <- i
			wg.Done()
		})
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if len(channel) != jobCount {
			t.Errorf("Expected channel length to be %d, but got %d", jobCount, len(channel))
		}
	case <-time.After(5 * time.Second):
		t.Error("Not all jobs were processed in time")
	}
}

func TestWorkerCount(t *testing.T) {
	p := pool.NewPool(POOL_WORKER_COUNT)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p.Run(ctx)

	workerCount := 0
	var mu sync.Mutex

	done := make(chan bool)
	for i := 0; i < POOL_WORKER_COUNT; i++ {
		p.AddJob(func() {
			mu.Lock()
			workerCount++
			mu.Unlock()
			done <- true
		})
	}

	for i := 0; i < POOL_WORKER_COUNT; i++ {
		<-done
	}

	if workerCount != POOL_WORKER_COUNT {
		t.Errorf("Expected %d workers to be running, but got %d", POOL_WORKER_COUNT, workerCount)
	}
}
