package raft

import (
	"context"
	"sync"
	"testing"
	"time"

	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/metrics"
	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// silenceSamples reads the leader silence histogram from the registry /metrics
// is served from.
func silenceSamples(t *testing.T) (count uint64, sum float64) {
	t.Helper()
	return histogramSamples(t, metricnames.RaftLeaderSilence)
}

// changeSamples is the same for the silence of a replaced leader.
func changeSamples(t *testing.T) (count uint64, sum float64) {
	t.Helper()
	return histogramSamples(t, metricnames.RaftLeaderChangeSilence)
}

func histogramSamples(t *testing.T, name string) (count uint64, sum float64) {
	t.Helper()
	series := metricFamily(t, name).GetMetric()
	require.Len(t, series, 1)
	histogram := series[0].GetHistogram()
	return histogram.GetSampleCount(), histogram.GetSampleSum()
}

func electionsStarted(t *testing.T, reason string) float64 {
	t.Helper()
	return metricSeries(t, metricnames.RaftElectionsStarted, metricnames.ReasonLabel, reason).GetCounter().GetValue()
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

func TestLeaderChangeSilence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const silence = 5 * time.Second
	// as if the leader's last message had arrived silence ago
	backdate := func(node *Raft) {
		node.leaderClock.Load().lastSeen.Store(int64(time.Since(leaderClockEpoch) - silence))
	}

	t.Run("a different leader records the old one's silence", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		first, second := testPeerID(t), testPeerID(t)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, first, 1)))
		backdate(node)

		start, sumBefore := changeSamples(t)
		silenceStart, _ := silenceSamples(t)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, second, 2)))
		require.Equal(t, second, node.GetLeader())

		count, sum := changeSamples(t)
		assert.Equal(t, start+1, count)
		assert.InDelta(t, silence.Seconds(), sum-sumBefore, 1)
		silenceCount, _ := silenceSamples(t)
		assert.Equal(t, silenceStart, silenceCount, "the other series stays out of it")
	})

	t.Run("this node winning records the old leader's silence", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, testPeerID(t), 1)))
		backdate(node)

		start, sumBefore := changeSamples(t)
		node.Mutex.Lock()
		node.setLeaderState()
		node.Mutex.Unlock()

		count, sum := changeSamples(t)
		assert.Equal(t, start+1, count)
		assert.InDelta(t, silence.Seconds(), sum-sumBefore, 1)
	})

	t.Run("measured from the old leader's last message", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		first := testPeerID(t)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, first, 1)))
		backdate(node)
		require.NoError(t, node.handleMessage(ctx, Message{Type: "trigger", SentFrom: first}))

		start, sumBefore := changeSamples(t)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, testPeerID(t), 2)))

		count, sum := changeSamples(t)
		assert.Equal(t, start+1, count)
		assert.Less(t, sum-sumBefore, silence.Seconds())
	})

	t.Run("no sample without a leader change", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		leader := testPeerID(t)

		start, _ := changeSamples(t)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 1)))
		count, _ := changeSamples(t)
		assert.Equal(t, start, count, "first leader replaces nobody")

		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 1)))
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 2)))
		count, _ = changeSamples(t)
		assert.Equal(t, start, count, "same peer again, even at a new term")
	})

	t.Run("no sample for same-term leaders flipping", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		first, second := testPeerID(t), testPeerID(t)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, first, 1)))

		start, _ := changeSamples(t)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, second, 1)))
		require.Equal(t, second, node.GetLeader())
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, first, 1)))
		require.Equal(t, first, node.GetLeader())
		count, _ := changeSamples(t)
		assert.Equal(t, start, count)

		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, second, 2)))
		count, _ = changeSamples(t)
		assert.Equal(t, start+1, count, "a higher term is a real change")
	})

	t.Run("no sample for a flip after the leader was re-elected", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		first, second := testPeerID(t), testPeerID(t)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, first, 1)))
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, first, 3)))

		start, _ := changeSamples(t)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, second, 3)))
		require.Equal(t, second, node.GetLeader())
		count, _ := changeSamples(t)
		assert.Equal(t, start, count)
	})

	t.Run("a silent leader replaced at the same term records", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		first, second := testPeerID(t), testPeerID(t)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, first, 3)))
		backdate(node)

		start, sumBefore := changeSamples(t)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, second, 3)))
		require.Equal(t, second, node.GetLeader())
		count, sum := changeSamples(t)
		assert.Equal(t, start+1, count)
		assert.InDelta(t, silence.Seconds(), sum-sumBefore, 1)
	})

	t.Run("a voter following the new leader at its own term records", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, testPeerID(t), 1)))

		// voting moved the term up before the new leader's first heartbeat
		node.Mutex.Lock()
		node.Term = 2
		node.Mutex.Unlock()

		start, _ := changeSamples(t)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, testPeerID(t), 2)))
		count, _ := changeSamples(t)
		assert.Equal(t, start+1, count)
	})

	t.Run("no sample when this node stops leading", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		node.Mutex.Lock()
		node.setLeaderState()
		term := node.Term
		node.Mutex.Unlock()

		start, _ := changeSamples(t)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, testPeerID(t), term+1)))
		require.Equal(t, Follower, node.GetRole())
		count, _ := changeSamples(t)
		assert.Equal(t, start, count)
	})
}

func TestLeaderSilenceIgnoresInvalidHeartbeats(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node := newSilenceTestNode(ctx, t)
	leader := testPeerID(t)
	require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 1)))
	lastSeen := node.leaderClock.Load().lastSeen.Load()
	start, _ := silenceSamples(t)

	mismatched := heartbeatFrom(t, testPeerID(t), 1)
	mismatched.SentFrom = leader
	require.ErrorIs(t, node.handleMessage(ctx, mismatched), errorSentinel.ErrRaftLeaderIdMismatch)

	require.Error(t, node.handleMessage(ctx, Message{Type: Heartbeat, SentFrom: leader, Data: []byte("{")}))

	count, _ := silenceSamples(t)
	assert.Equal(t, start, count)
	assert.Equal(t, lastSeen, node.leaderClock.Load().lastSeen.Load())
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
