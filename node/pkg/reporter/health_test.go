package reporter

import (
	"context"
	"testing"
	"time"

	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthEvaluate(t *testing.T) {
	t0 := time.UnixMilli(1_700_000_000_000)
	past := func(d time.Duration) time.Time { return t0.Add(d) }

	t.Run("not running", func(t *testing.T) {
		h := newHealth(t0)
		ok, _ := h.evaluate(past(unhealthyAfter))
		assert.True(t, ok, "unhealthy within the start grace")
		ok, reason := h.evaluate(past(unhealthyAfter + time.Second))
		assert.False(t, ok)
		assert.Equal(t, "not running", reason)
		h.setRunning()
		ok, _ = h.evaluate(past(unhealthyAfter + time.Second))
		assert.True(t, ok)
	})

	t.Run("submissions failing", func(t *testing.T) {
		h := newHealth(t0)
		h.setRunning()
		h.end(h.begin(t0), t0, false)
		h.end(h.begin(past(time.Second)), past(time.Second), false)
		ok, _ := h.evaluate(past(unhealthyAfter))
		assert.True(t, ok, "failing since the first failure, not the last")
		ok, reason := h.evaluate(past(unhealthyAfter + time.Second))
		assert.False(t, ok)
		assert.Equal(t, "submissions failing", reason)

		// one that gets through clears it
		h.end(h.begin(past(unhealthyAfter)), past(unhealthyAfter), true)
		ok, _ = h.evaluate(past(2 * unhealthyAfter))
		assert.True(t, ok)
	})

	t.Run("submission stuck", func(t *testing.T) {
		h := newHealth(t0)
		h.setRunning()
		id := h.begin(t0)
		h.end(h.begin(past(time.Second)), past(time.Second), true)
		ok, reason := h.evaluate(past(unhealthyAfter + time.Second))
		assert.False(t, ok, "a later delivered report must not hide a hung one")
		assert.Equal(t, "submission stuck", reason)
		h.end(id, past(unhealthyAfter+time.Second), true)
		ok, _ = h.evaluate(past(unhealthyAfter + time.Second))
		assert.True(t, ok)
	})
}

func TestHealthyStandaloneAlwaysTrue(t *testing.T) {
	assert.True(t, New().Healthy())
}

func reporterUpValue(t *testing.T) (float64, bool) {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, f := range families {
		if f.GetName() == metricnames.ReporterUp {
			return f.GetMetric()[0].GetGauge().GetValue(), true
		}
	}
	return 0, false
}

// A reporter that can't start is retried, and turns unhealthy (Healthy, up
// gauge) once past unhealthyAfter, so the node hands its feeds over (#2914)
// and the failure is alertable (#2895).
func TestRunWithRetryUnhealthyWhileNotStarted(t *testing.T) {
	t.Setenv("SUBMISSION_PROXY_CONTRACT", "") // Run fails before any network call
	prevAfter, prevCheck, prevMin := unhealthyAfter, healthCheckInterval, runRetryMin
	unhealthyAfter, healthCheckInterval, runRetryMin = 100*time.Millisecond, 10*time.Millisecond, 10*time.Millisecond
	t.Cleanup(func() { unhealthyAfter, healthCheckInterval, runRetryMin = prevAfter, prevCheck, prevMin })

	ctx, cancel := context.WithCancel(context.Background())
	a := NewNodeApp()
	assert.True(t, a.Healthy(), "unhealthy before the start grace")

	done := make(chan error, 1)
	go func() { done <- a.RunWithRetry(ctx) }()

	require.Eventually(t, func() bool { return !a.Healthy() }, 5*time.Second, 10*time.Millisecond)
	up, ok := reporterUpValue(t)
	require.True(t, ok, "no up gauge")
	assert.Equal(t, 0.0, up)

	select {
	case err := <-done:
		t.Fatalf("RunWithRetry returned while Run kept failing: %v", err)
	default:
	}
	cancel()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("RunWithRetry did not return on cancel")
	}
}
