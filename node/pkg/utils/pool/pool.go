package pool

import (
	"context"
)

type Pool struct {
	jobChannel  chan func()
	workerCount int
	ctx         context.Context
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
	p.ctx = poolCtx
	p.Cancel = cancel
	p.IsRunning = true

	for i := 0; i < p.workerCount; i++ {
		go p.worker()
	}
}

func (p *Pool) worker() {
	for {
		select {
		case job := <-p.jobChannel:
			job()
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *Pool) AddJob(job func()) {
	select {
	case p.jobChannel <- job:
		return
	case <-p.ctx.Done():
		return
	}
}
