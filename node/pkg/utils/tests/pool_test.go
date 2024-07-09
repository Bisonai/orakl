package tests

import (
	"context"
	"testing"
	"time"

	pool "bisonai.com/orakl/node/pkg/utils/pool"
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
	p.Run(ctx)
	defer cancel()

	var slice []int
	var confirm_slice []int
	for i := 0; i < 10; i++ {
		confirm_slice = append(confirm_slice, i)
		p.AddJob(func() {
			slice = append(slice, i)
		})
		time.Sleep(100 * time.Millisecond)
	}

	for i := 0; i < 10; i++ {
		if slice[i] != confirm_slice[i] {
			t.Errorf("Job execution failed")
		}
	}
}
