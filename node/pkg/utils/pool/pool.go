package pool

import (
	"context"
)

type Pool struct {
	jobChannel chan func()
}

const POOL_WORKER_COUNT = 3

func NewPool() *Pool {
	return &Pool{
		jobChannel: make(chan func()),
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

func (p *Pool) AddJob(job func()) {
	p.jobChannel <- job
}
