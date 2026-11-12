package aggregator

import (
	"context"
	"sync"
)

// WaitStopped blocks until every aggregator's Run has returned, or ctx is done,
// and from then on no aggregator starts. Cancelling the context given to Run
// stops them all: each Raft resigns the leader role, and Run returns only after
// the leader loop and its jobs have (raft.Run), so no round follows. For
// shutdown.
func (a *App) WaitStopped(ctx context.Context) error {
	return a.runs.closeAndWait(ctx)
}

// runTracker counts running aggregator Run goroutines and, once closed, refuses
// new ones, so a shutdown during startup can't be followed by a start. The
// zero value is ready.
type runTracker struct {
	mu     sync.Mutex
	n      int
	idle   chan struct{} // closed when n drops to 0; nil while none ran
	closed bool
}

// start counts one more Run, unless the tracker is closed.
func (t *runTracker) start() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return false
	}
	if t.n == 0 {
		t.idle = make(chan struct{})
	}
	t.n++
	return true
}

func (t *runTracker) stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.n--
	if t.n == 0 {
		close(t.idle)
	}
}

// closeAndWait refuses further starts and waits for running ones.
func (t *runTracker) closeAndWait(ctx context.Context) error {
	t.mu.Lock()
	t.closed = true
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
