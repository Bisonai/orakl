package pool

import (
	"context"
)

type Pool struct {
	jobChannel  chan func()
	workerCount int
}

func NewPool(workerCount int) *Pool {
	return &Pool{
		jobChannel:  make(chan func()),
		workerCount: workerCount,
	}
}

func (p *Pool) Run(ctx context.Context) {
	for i := 0; i < p.workerCount; i++ {
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
