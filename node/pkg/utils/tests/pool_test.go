package tests

import (
	"context"
	"testing"
	"time"

	goroutine_pool "bisonai.com/orakl/node/pkg/utils/goroutine-pool"
)

func TestNewPool(t *testing.T) {
	pool := goroutine_pool.NewPool()
	if pool == nil {
		t.Errorf("NewPool() returned nil")
	}
}

func TestJobExecution(t *testing.T) {
	pool := goroutine_pool.NewPool()
	ctx, cancel := context.WithCancel(context.Background())
	pool.Run(ctx)
	defer cancel()

	var slice []int
	var confirm_slice []int
	for i := 0; i < 10; i++ {
		confirm_slice = append(confirm_slice, i)
		pool.AddJob(func() {
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
