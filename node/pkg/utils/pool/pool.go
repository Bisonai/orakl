package pool

import (
	"context"
)

type Pool struct {
	jobChannel  chan func()
	workerCount int
	Cancel      context.CancelFunc
	IsRunning   bool
}

func NewPool(workerCount int) *Pool {
	return &Pool{
		jobChannel:  make(chan func()),
		workerCount: workerCount,
	}
}

func (p *Pool) Run(ctx context.Context) {
	poolCtx, cancel := context.WithCancel(ctx)
	p.Cancel = cancel
	p.IsRunning = true

	for i := 0; i < p.workerCount; i++ {
		go p.worker(poolCtx)
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
	if !p.IsRunning {
		return
	}
	p.jobChannel <- job
}
