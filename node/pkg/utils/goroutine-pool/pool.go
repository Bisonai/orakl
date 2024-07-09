package goroutine_pool

import (
	"context"
)

type PoolJob func()

type Pool struct {
	jobChannel chan PoolJob
}

const POOL_WORKER_COUNT = 3

func NewPool() *Pool {
	return &Pool{
		jobChannel: make(chan PoolJob),
	}
}

func (p *Pool) Run(ctx context.Context) {
	for i := 0; i < POOL_WORKER_COUNT; i++ {
		go p.worker(ctx)
	}
}

func (p *Pool) worker(ctx context.Context) {
	for {
		select {
		case job := <-p.jobChannel:
			job()
		case <-ctx.Done():
			return
		}
	}
}

func (p *Pool) AddJob(job PoolJob) {
	p.jobChannel <- job
}
