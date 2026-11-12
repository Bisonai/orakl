package aggregator

import (
	"context"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/raft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Cancelling the aggregators' context (shutdown) resigns the leader, and once
// WaitStopped returns no round is produced.
func TestShutdownResignsAndStopsRounds(t *testing.T) {
	ctx := context.Background()
	app, configID := newLifecycleTestApp(t, ctx)
	configs, err := app.getConfigs(ctx)
	require.NoError(t, err)
	require.NoError(t, app.setAggregators(ctx, app.Host, app.Pubsub, configs))
	agg := app.Aggregators[configID]
	roundID := func() int32 {
		agg.mu.RLock()
		defer agg.mu.RUnlock()
		return agg.RoundID
	}

	runCtx, stopAll := context.WithCancel(ctx)
	defer stopAll()
	require.NoError(t, app.startAllAggregators(runCtx))
	require.Eventually(t, func() bool { return agg.Raft.GetRole() == raft.Leader }, 15*time.Second, 50*time.Millisecond, "lone node never led")
	led := roundID()
	require.Eventually(t, func() bool { return roundID() > led }, 15*time.Second, 50*time.Millisecond, "no rounds")

	stopAll()
	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	require.NoError(t, app.WaitStopped(waitCtx))
	assert.NotEqual(t, raft.Leader, agg.Raft.GetRole(), "still leader after shutdown")

	last := roundID()
	time.Sleep(3 * time.Duration(agg.AggregateInterval) * time.Millisecond)
	assert.Equal(t, last, roundID(), "round produced after shutdown")
}

func TestRunTrackerWait(t *testing.T) {
	var tracker runTracker
	require.True(t, tracker.start())
	require.True(t, tracker.start())
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	assert.ErrorIs(t, tracker.closeAndWait(ctx), context.DeadlineExceeded)

	tracker.stop()
	done := make(chan error, 1)
	go func() { done <- tracker.closeAndWait(context.Background()) }()
	tracker.stop()
	require.NoError(t, <-done)
}

// A shutdown during startup: nothing ran yet when the wait returns, and no
// aggregator starts after it.
func TestRunTrackerRefusesStartAfterWait(t *testing.T) {
	var tracker runTracker
	require.NoError(t, tracker.closeAndWait(context.Background()), "nothing ran")
	assert.False(t, tracker.start(), "started after shutdown")
}
