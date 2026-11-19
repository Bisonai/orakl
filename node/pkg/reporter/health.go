package reporter

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"bisonai.com/miko/node/pkg/metrics/submission"
	"github.com/rs/zerolog/log"
)

// Variables for tests.
var (
	// unhealthyAfter is how long the in-node reporter may be not running,
	// failing every submission, or stuck in one, before it counts as
	// unhealthy and the node gives up leading feeds (#2914).
	unhealthyAfter = 30 * time.Second
	// healthCheckInterval is how often Healthy and the up gauge are updated.
	healthCheckInterval = time.Second

	// RunWithRetry's wait after a failed start, doubling up to the max.
	runRetryMin = 5 * time.Second
	runRetryMax = time.Minute
)

// health tracks whether the in-node reporter can submit.
type health struct {
	// unhealthyAfter and healthCheckInterval when created
	after, interval time.Duration

	mu sync.Mutex
	// when the reporter was created; it counts as not running from here
	// until Run succeeds
	createdAt time.Time
	running   bool
	// first report since the last one that got through in which every batch
	// failed; zero: the last report got through (or none yet)
	failingSince time.Time
	// start of each report still in flight, by id
	inflight map[uint64]time.Time
	nextID   uint64

	// last evaluate result; true until the first evaluation
	ok atomic.Bool
}

func newHealth(now time.Time) *health {
	h := &health{after: unhealthyAfter, interval: healthCheckInterval, createdAt: now, inflight: map[uint64]time.Time{}}
	h.ok.Store(true)
	return h
}

func (h *health) setRunning() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.running = true
}

// begin registers a report starting at now; end(id) must follow.
func (h *health) begin(now time.Time) uint64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nextID++
	h.inflight[h.nextID] = now
	return h.nextID
}

// end finishes report id; delivered is whether any of its batches got through.
func (h *health) end(id uint64, now time.Time, delivered bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.inflight, id)
	if delivered {
		h.failingSince = time.Time{}
	} else if h.failingSince.IsZero() {
		h.failingSince = now
	}
}

// evaluate reports whether the reporter is healthy at now, and if not why:
// it is not running, every submission failed, or one is stuck, for longer
// than unhealthyAfter.
func (h *health) evaluate(now time.Time) (bool, string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.running && now.Sub(h.createdAt) > h.after {
		return false, "not running"
	}
	if !h.failingSince.IsZero() && now.Sub(h.failingSince) > h.after {
		return false, "submissions failing"
	}
	for _, start := range h.inflight {
		if now.Sub(start) > h.after {
			return false, "submission stuck"
		}
	}
	return true, ""
}

// check evaluates at now and publishes the result to Healthy and the gauge,
// logging changes.
func (h *health) check(now time.Time) {
	ok, reason := h.evaluate(now)
	if prev := h.ok.Swap(ok); prev != ok {
		if ok {
			log.Info().Str("Player", "Reporter").Msg("in-node reporter healthy again")
		} else {
			log.Warn().Str("Player", "Reporter").Str("reason", reason).Dur("after", h.after).
				Msg("in-node reporter unhealthy, handing its feeds to peers")
		}
	}
	submission.SetReporterUp(ok)
}

func (h *health) watch(ctx context.Context, now func() time.Time) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()
	for {
		h.check(now())
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Healthy reports whether this node's in-node reporter can submit: false once
// it has not been running, or every submission failed or one hung, for longer
// than unhealthyAfter. Updated by RunWithRetry. Always true for a standalone
// reporter.
func (a *App) Healthy() bool {
	if a.local == nil {
		return true
	}
	return a.local.health.ok.Load()
}

// RunWithRetry runs Run until it succeeds or ctx is done, waiting runRetryMin
// after a failure, doubling up to runRetryMax. For the in-node reporter it
// also keeps Healthy and the orakl_node_reporter_up gauge current. It blocks
// until Run succeeded, or ctx is done.
func (a *App) RunWithRetry(ctx context.Context) error {
	if a.local != nil {
		submission.RegisterReporterUp()
		go a.local.health.watch(ctx, a.local.now)
	}
	wait := runRetryMin
	for {
		err := a.Run(ctx)
		if err == nil {
			return nil
		}
		log.Error().Str("Player", "Reporter").Err(err).Dur("retryIn", wait).Msg("failed to start reporter, retrying")
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
		wait = min(2*wait, runRetryMax)
	}
}
