package raft

import (
	"context"
	"sync"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/metrics"
	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// silenceSamples reads the leader silence histogram from the registry /metrics
// is served from.
func silenceSamples(t *testing.T) (count uint64, sum float64) {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() == metricnames.RaftLeaderSilence {
			histogram := family.GetMetric()[0].GetHistogram()
			return histogram.GetSampleCount(), histogram.GetSampleSum()
		}
	}
	t.Fatalf("%s not registered", metricnames.RaftLeaderSilence)
	return 0, 0
}

func electionsStarted(t *testing.T, reason string) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != metricnames.RaftElectionsStarted {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == metricnames.ReasonLabel && label.GetValue() == reason {
					return metric.GetCounter().GetValue()
				}
			}
		}
	}
	t.Fatalf("%s{%s=%q} not registered", metricnames.RaftElectionsStarted, metricnames.ReasonLabel, reason)
	return 0
}

func heartbeatFrom(t *testing.T, leaderID string, term int) Message {
	t.Helper()
	data, err := encodeInner(HeartbeatMessage{LeaderID: leaderID, Term: term})
	require.NoError(t, err)
	return Message{Type: Heartbeat, SentFrom: leaderID, Data: data}
}

func newSilenceTestNode(ctx context.Context, t *testing.T) *Raft {
	t.Helper()
	node := newPublishTestNode(ctx, t)
	node.startElectionTimer()
	node.HandleCustomMessage = func(context.Context, Message) error { return nil }
	return node
}

func TestLeaderSilence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const gap = 20 * time.Millisecond
	customFrom := func(sentFrom string) Message {
		return Message{Type: "trigger", SentFrom: sentFrom}
	}

	t.Run("observed for leader messages of any type", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		leader := testPeerID(t)

		start, sumBefore := silenceSamples(t)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 1)))
		require.Equal(t, leader, node.GetLeader())
		count, _ := silenceSamples(t)
		assert.Equal(t, start, count, "message that makes the leader")

		time.Sleep(gap)
		require.NoError(t, node.handleMessage(ctx, customFrom(leader)))
		count, sum := silenceSamples(t)
		assert.Equal(t, start+1, count, "custom message")
		assert.GreaterOrEqual(t, sum-sumBefore, gap.Seconds())

		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 1)))
		count, _ = silenceSamples(t)
		assert.Equal(t, start+2, count, "heartbeat")
	})

	t.Run("other peers do not count or move the clock", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		leader := testPeerID(t)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 1)))
		require.NoError(t, node.handleMessage(ctx, customFrom(leader)))
		lastSeen := node.leaderClock.Load().lastSeen.Load()

		start, _ := silenceSamples(t)
		require.NoError(t, node.handleMessage(ctx, customFrom(testPeerID(t))))
		count, _ := silenceSamples(t)
		assert.Equal(t, start, count)
		assert.Equal(t, lastSeen, node.leaderClock.Load().lastSeen.Load())
	})

	t.Run("own messages do not count", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		self := node.GetHostId()
		leader := testPeerID(t)

		start, _ := silenceSamples(t)
		for i := 0; i < 3; i++ {
			require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, self, 1)))
			require.NoError(t, node.handleMessage(ctx, customFrom(self)))
		}
		count, _ := silenceSamples(t)
		assert.Equal(t, start, count, "follower without a leader")

		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 1)))
		for i := 0; i < 3; i++ {
			require.NoError(t, node.handleMessage(ctx, customFrom(self)))
		}
		count, _ = silenceSamples(t)
		assert.Equal(t, start, count, "follower with a leader")

		node.Mutex.Lock()
		node.setLeaderState()
		node.Mutex.Unlock()
		for i := 0; i < 3; i++ {
			require.NoError(t, node.handleMessage(ctx, customFrom(self)))
			require.NoError(t, node.handleMessage(ctx, customFrom(leader)))
		}
		count, _ = silenceSamples(t)
		assert.Equal(t, start, count, "leader")
	})

	t.Run("first message after a leader change is skipped", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		first, second := testPeerID(t), testPeerID(t)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, first, 1)))

		start, sumBefore := silenceSamples(t)
		time.Sleep(10 * gap)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, second, 2)))
		require.Equal(t, second, node.GetLeader())
		count, _ := silenceSamples(t)
		assert.Equal(t, start, count)

		// measured from the new leader's first message, not across the change
		require.NoError(t, node.handleMessage(ctx, customFrom(second)))
		count, sum := silenceSamples(t)
		assert.Equal(t, start+1, count)
		assert.Less(t, sum-sumBefore, (10 * gap).Seconds())

		// the old leader isn't current anymore
		require.NoError(t, node.handleMessage(ctx, customFrom(first)))
		count, _ = silenceSamples(t)
		assert.Equal(t, start+1, count)
	})

	t.Run("does not change what a heartbeat does", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		leader := testPeerID(t)
		node.MissedHeartbeats = 1

		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 4)))

		assert.Equal(t, 4, node.GetCurrentTerm())
		assert.Equal(t, Follower, node.GetRole())
		assert.Equal(t, leader, node.GetLeader())
		assert.Equal(t, 0, node.MissedHeartbeats)

		// a custom message still doesn't count as liveness
		node.MissedHeartbeats = 1
		require.NoError(t, node.handleMessage(ctx, customFrom(leader)))
		assert.Equal(t, 1, node.MissedHeartbeats)
	})

	t.Run("concurrent handlers", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		leader := testPeerID(t)
		heartbeat := heartbeatFrom(t, leader, 1)

		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				if i%2 == 0 {
					assert.NoError(t, node.handleMessage(ctx, heartbeat))
				} else {
					assert.NoError(t, node.handleMessage(ctx, customFrom(leader)))
				}
			}(i)
		}
		wg.Wait()
		assert.Equal(t, leader, node.GetLeader())
	})
}

func TestRecordLeaderSilenceClockNeverMovesBack(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node := newSilenceTestNode(ctx, t)
	leader := testPeerID(t)
	require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 1)))

	// as if a newer message's handler had already run
	clock := node.leaderClock.Load()
	ahead := int64(time.Since(leaderClockEpoch) + time.Hour)
	clock.lastSeen.Store(ahead)

	start, _ := silenceSamples(t)
	node.recordLeaderSilence(leader)
	count, _ := silenceSamples(t)
	assert.Equal(t, start, count)
	assert.Equal(t, ahead, clock.lastSeen.Load())
}

func TestRecordLeaderSilenceDoesNotAllocate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node := newSilenceTestNode(ctx, t)
	leader := testPeerID(t)
	require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 1)))

	assert.Zero(t, testing.AllocsPerRun(100, func() { node.recordLeaderSilence(leader) }))
}

func TestStartElectionCountsOnlyStartedElections(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node := newSilenceTestNode(ctx, t)
	timeout := metrics.RaftElectionReasonTimeout
	start := electionsStarted(t, timeout)

	// below the missed heartbeat threshold
	node.startElection(ctx)
	assert.Equal(t, start, electionsStarted(t, timeout), "missed heartbeat only")
	assert.Equal(t, Follower, node.GetRole())

	node.MissedHeartbeats = MaxMissedHeartbeats
	node.startElection(ctx)
	assert.Equal(t, start+1, electionsStarted(t, timeout))
	assert.Equal(t, Candidate, node.GetRole())

	// in cooldown
	node.MissedHeartbeats = MaxMissedHeartbeats
	node.startElection(ctx)
	assert.Equal(t, start+1, electionsStarted(t, timeout), "cooldown")
}
