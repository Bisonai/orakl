package reporter

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"bisonai.com/orakl/node/pkg/metrics/submission"
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

// health tracks whether the in-node reporter can submit. Submissions are
// tracked per regular reporter (interval group), so a group that delivers
// can't hide one that fails; the deviation reporter isn't tracked: it shares
// the regular groups' chain helper, and finding no deviation can't be told
// apart from leading nothing.
type health struct {
	// unhealthyAfter and healthCheckInterval when created
	after, interval time.Duration

	mu sync.Mutex
	// when the reporter was created; it counts as not running from here
	// until Run succeeds
	createdAt time.Time
	running   bool
	groups    map[any]*groupHealth
	// stopped groups: a job still running when its group stopped must not
	// bring its state back
	forgotten map[any]bool
	nextID    uint64

	// last evaluate result; true until the first evaluation
	ok atomic.Bool
}

type groupHealth struct {
	// first failed report since the last delivered one; zero: none
	failingSince time.Time
	// start of the last delivered report: an older one failing late doesn't
	// count, jobs overlap
	deliveredStart time.Time
	// start of each report still in flight, by id
	inflight map[uint64]time.Time
}

func newHealth(now time.Time) *health {
	h := &health{after: unhealthyAfter, interval: healthCheckInterval, createdAt: now, groups: map[any]*groupHealth{}, forgotten: map[any]bool{}}
	h.ok.Store(true)
	return h
}

func (h *health) setRunning() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.running = true
}

// group returns group's state, created on first use; nil once forgotten.
// Caller holds mu.
func (h *health) group(group any) *groupHealth {
	if h.forgotten[group] {
		return nil
	}
	g, ok := h.groups[group]
	if !ok {
		g = &groupHealth{inflight: map[uint64]time.Time{}}
		h.groups[group] = g
	}
	return g
}

// forget drops a stopped group, so its last state can't keep the node
// unhealthy.
func (h *health) forget(group any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.groups, group)
	h.forgotten[group] = true
}

// begin registers a report of group starting at now; end(id) must follow.
func (h *health) begin(group any, now time.Time) uint64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nextID++
	if g := h.group(group); g != nil {
		g.inflight[h.nextID] = now
	}
	return h.nextID
}

// end finishes report id of group; delivered is whether any of its batches got
// through. A delivered report also drops the group's reports in flight since
// before it started: one hung call must not keep a working group unhealthy.
func (h *health) end(group any, id uint64, now time.Time, delivered bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	g, ok := h.groups[group]
	if !ok {
		return // forgotten
	}
	start, ok := g.inflight[id]
	if !ok {
		return // dropped: superseded by a delivered report, or the group went idle
	}
	delete(g.inflight, id)
	if !delivered {
		if g.failingSince.IsZero() && start.After(g.deliveredStart) {
			g.failingSince = now
		}
		return
	}
	g.failingSince = time.Time{}
	g.deliveredStart = start
	for other, at := range g.inflight {
		if !at.After(start) {
			delete(g.inflight, other)
		}
	}
}

// prepared records a regular job's prepare for group: led pairs this node led
// with fresh data, ready of them left to submit. Led pairs with none left
// (e.g. every proof rejected) count as a failed report. Leading none clears
// the group's failure and reports in flight: it has nothing to submit, and
// must not stay unhealthy until it leads again.
func (h *health) prepared(group any, now time.Time, led, ready int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	g := h.group(group)
	if g == nil {
		return
	}
	switch {
	case led == 0:
		g.failingSince = time.Time{}
		clear(g.inflight)
	case ready == 0 && g.failingSince.IsZero():
		g.failingSince = now
	}
}

// evaluate reports whether the reporter is healthy at now, and if not why:
// it is not running, or a group's submissions all failed or one is stuck,
// for longer than unhealthyAfter. running is whether Run succeeded.
func (h *health) evaluate(now time.Time) (ok bool, reason string, running bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.running && now.Sub(h.createdAt) > h.after {
		return false, "not running", false
	}
	for _, g := range h.groups {
		if !g.failingSince.IsZero() && now.Sub(g.failingSince) > h.after {
			return false, "submissions failing", h.running
		}
		for _, start := range g.inflight {
			if now.Sub(start) > h.after {
				return false, "submission stuck", h.running
			}
		}
	}
	return true, "", h.running
}

// check evaluates at now and publishes the result to Healthy and the gauge,
// logging changes.
func (h *health) check(now time.Time) {
	ok, reason, running := h.evaluate(now)
	if prev := h.ok.Swap(ok); prev != ok {
		if ok {
			log.Info().Str("Player", "Reporter").Msg("in-node reporter healthy again")
		} else {
			log.Warn().Str("Player", "Reporter").Str("reason", reason).Dur("after", h.after).
				Msg("in-node reporter unhealthy, handing its feeds to peers")
		}
	}
	// Healthy has a start grace, the gauge none: up only once running
	submission.SetReporterUp(ok && running)
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
