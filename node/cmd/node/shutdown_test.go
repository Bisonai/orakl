package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type stepLog struct {
	mu    sync.Mutex
	steps []string
}

func (l *stepLog) add(step string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.steps = append(l.steps, step)
}

func (l *stepLog) get() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.steps...)
}

// The hand-over closes the host right after stopping the aggregators, so peers
// elect while they stop, and flushes rounds last, once nothing can advance them.
func TestHandOverOrder(t *testing.T) {
	var log stepLog
	stopped := make(chan struct{})
	handOver(shutdownSteps{
		stopAggregators: func() { log.add("stop"); close(stopped) },
		closeHost:       func() error { log.add("close"); return nil },
		waitAggregators: func(ctx context.Context) error {
			<-stopped
			log.add("wait")
			return nil
		},
		flushRounds: func(context.Context) error { log.add("flush"); return nil },
	})
	assert.Equal(t, []string{"stop", "close", "wait", "flush"}, log.get())
}

// A step that hangs is cut at its bound, and the rest still run: rounds are
// persisted even if aggregators or the host don't stop.
func TestHandOverBoundsHangingSteps(t *testing.T) {
	for name, set := range map[string]*time.Duration{"resign": &resignTimeout, "closeHost": &closeHostTimeout} {
		t.Run(name, func(t *testing.T) {
			saved := *set
			*set = 50 * time.Millisecond
			defer func() { *set = saved }()

			var log stepLog
			hang := make(chan struct{})
			defer close(hang)
			start := time.Now()
			handOver(shutdownSteps{
				stopAggregators: func() { log.add("stop") },
				waitAggregators: func(ctx context.Context) error {
					if name == "resign" {
						<-ctx.Done()
						return ctx.Err()
					}
					log.add("wait")
					return nil
				},
				closeHost: func() error {
					if name == "closeHost" {
						<-hang
					}
					log.add("close")
					return nil
				},
				flushRounds: func(context.Context) error { log.add("flush"); return nil },
			})
			assert.Less(t, time.Since(start), time.Second)
			assert.Contains(t, log.get(), "flush")
		})
	}
}
