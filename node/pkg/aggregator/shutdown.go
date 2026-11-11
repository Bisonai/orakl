package aggregator

import (
	"context"
	"sync"
)

// WaitStopped blocks until every aggregator's Run has returned, or ctx is done.
// Cancelling the context given to Run stops them all: each Raft resigns the
// leader role, and Run returns only after the leader loop and its jobs have
// (raft.Run), so no round follows. For shutdown.
func (a *App) WaitStopped(ctx context.Context) error {
	return a.runs.wait(ctx)
}

// runTracker counts running aggregator Run goroutines. The zero value is ready.
// Unlike a WaitGroup, a start may come while a wait is pending (a stop during
// startup), and the wait then also covers it.
type runTracker struct {
	mu   sync.Mutex
	n    int
	idle chan struct{} // closed when n drops to 0; nil while none ran
}

func (t *runTracker) start() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.n == 0 {
		t.idle = make(chan struct{})
	}
	t.n++
}

func (t *runTracker) stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.n--
	if t.n == 0 {
		close(t.idle)
	}
}

func (t *runTracker) wait(ctx context.Context) error {
	t.mu.Lock()
	if t.n == 0 {
		t.mu.Unlock()
		return nil
	}
	idle := t.idle
	t.mu.Unlock()
	select {
	case <-idle:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
