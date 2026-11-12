package raft

import (
	"context"
	"fmt"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain keeps a developer's P2P_SPARSE_HEARTBEAT out of the tests that
// build nodes with NewRaftNode and expect a heartbeat every tick.
func TestMain(m *testing.M) {
	_ = os.Unsetenv(SparseHeartbeatEnvVar)
	os.Exit(m.Run())
}

// priceFixDelay is how long after a trigger the leader sends its price fix in
// these tests: on two nodes, one round trip for the follower's price.
const priceFixDelay = 5 * time.Millisecond

// testSparseHB is the heartbeat interval in these tests: half the default, so
// every derived bound (stickiness window, skip window, maxHeartbeatGap,
// election timer) is half as long, with room left for a loaded runner.
const testSparseHB = HEARTBEAT_TIMEOUT / 2

var testLeaderTypes = map[MessageType]bool{"trigger": true, "priceFix": true}

func sparseTimers(r *Raft) {
	r.HeartbeatTimeout = testSparseHB
	r.CooldownPeriod = 10 * testSparseHB
	r.LeaderMessageTypes = testLeaderTypes
}

func sparseOn(r *Raft) {
	sparseTimers(r)
	r.SparseHeartbeat = true
}

func sparseOff(r *Raft) {
	sparseTimers(r)
	r.SparseHeartbeat = false
}

// withRounds makes the node's leader job an aggregator round every interval:
// a trigger, then a price fix priceFixDelay later. interval 0 sends nothing,
// like a feed whose rounds are far apart.
func withRounds(interval time.Duration) func(*Raft) {
	return func(r *Raft) {
		if interval == 0 {
			r.LeaderJobTimeout = time.Hour
			r.LeaderJob = func(context.Context) error { return nil }
			return
		}
		r.LeaderJobTimeout = interval
		r.LeaderJob = func(ctx context.Context) error {
			if err := r.PublishMessage(ctx, leaderMessageFrom(r.GetHostId(), "trigger")); err != nil {
				return err
			}
			time.AfterFunc(priceFixDelay, func() {
				if ctx.Err() == nil {
					_ = r.PublishMessage(ctx, leaderMessageFrom(r.GetHostId(), "priceFix"))
				}
			})
			return nil
		}
	}
}

// leaderSends records when the messages a node publishes arrive back on its
// own subscription, by type.
type leaderSends struct {
	mu    sync.Mutex
	times map[MessageType][]time.Time
}

func recordSends(ctx context.Context, t *testing.T, node *Raft) *leaderSends {
	t.Helper()
	sub, err := node.Topic.Subscribe()
	require.NoError(t, err)
	t.Cleanup(sub.Cancel)

	sends := &leaderSends{times: map[MessageType][]time.Time{}}
	go func() {
		for {
			raw, err := sub.Next(ctx)
			if err != nil {
				return
			}
			msg, err := node.unmarshalMessage(raw.Data)
			if err != nil {
				continue
			}
			sends.mu.Lock()
			sends.times[msg.Type] = append(sends.times[msg.Type], time.Now())
			sends.mu.Unlock()
		}
	}()
	return sends
}

func (s *leaderSends) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.times = map[MessageType][]time.Time{}
}

func (s *leaderSends) count(msgType MessageType) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.times[msgType])
}

// maxGap is the longest gap between consecutive messages of the given types.
func (s *leaderSends) maxGap(types ...MessageType) time.Duration {
	s.mu.Lock()
	var all []time.Time
	for _, msgType := range types {
		all = append(all, s.times[msgType]...)
	}
	s.mu.Unlock()
	slices.SortFunc(all, time.Time.Compare)
	var gap time.Duration
	for i := 1; i < len(all); i++ {
		gap = max(gap, all[i].Sub(all[i-1]))
	}
	return gap
}

func heartbeatsSkipped(t *testing.T) float64 {
	t.Helper()
	return metricFamily(t, metricnames.RaftHeartbeatsSkipped).GetMetric()[0].GetCounter().GetValue()
}

// startLoneLeader runs a single node, which leads on its own vote.
func startLoneLeader(ctx context.Context, t *testing.T, configure ...func(*Raft)) (*Raft, *leaderSends) {
	t.Helper()
	nodes := newCluster(ctx, t, 1, configure...)
	sends := recordSends(ctx, t, nodes[0])
	go nodes[0].Run(ctx)
	waitForSingleLeader(t, nodes, "lone node did not lead")
	return nodes[0], sends
}

func TestSparseHeartbeatEnv(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for value, want := range map[string]bool{"": false, "false": false, "0": false, "yes": false, "true": true, "1": true, " TRUE ": true} {
		t.Setenv(SparseHeartbeatEnvVar, value)
		assert.Equal(t, want, SparseHeartbeatEnabled(), "%q", value)
		assert.Equal(t, want, newPublishTestNode(ctx, t).SparseHeartbeat, "%q", value)
	}
}

// With the flag on, a 400 ms feed (4 heartbeat intervals) sends one heartbeat a
// round instead of four. Off, every tick sends one and nothing is skipped.
func TestSparseHeartbeatCount(t *testing.T) {
	const window = 60 * testSparseHB
	counts := map[bool]int{}
	for _, on := range []bool{false, true} {
		t.Run(fmt.Sprintf("sparse=%v", on), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			flag := sparseOff
			if on {
				flag = sparseOn
			}
			node, sends := startLoneLeader(ctx, t, flag, withRounds(4*testSparseHB))
			time.Sleep(testSparseHB)
			sends.reset()
			skipped := heartbeatsSkipped(t)
			time.Sleep(window)

			counts[on] = sends.count(Heartbeat)
			rounds := sends.count("trigger")
			t.Logf("heartbeats %d, rounds %d, skipped %v", counts[on], rounds, heartbeatsSkipped(t)-skipped)
			require.Positive(t, rounds)
			if on {
				assert.Positive(t, heartbeatsSkipped(t)-skipped)
				assert.LessOrEqual(t, counts[on], 2*rounds, "more heartbeats than rounds leave room for")
			} else {
				assert.Zero(t, node.leaderSentAt.Load(), "sends stamped with the flag off")
				assert.Zero(t, node.heartbeatSentAt.Load(), "sends stamped with the flag off")
				assert.GreaterOrEqual(t, counts[on], int(window/testSparseHB)*8/10, "a heartbeat missing with the flag off")
			}
		})
	}
	assert.LessOrEqual(t, counts[true]*100, counts[false]*40, "heartbeats down by less than 60%")
}

// The longest gap between leader messages (heartbeats, triggers, price fixes)
// stays below half the stickiness window plus a tick, whatever the round
// interval, and heartbeats are never further apart than maxHeartbeatGap.
func TestSparseHeartbeatLeaderGapBound(t *testing.T) {
	for _, interval := range []time.Duration{0, testSparseHB, 4 * testSparseHB, 7 * testSparseHB, 50 * testSparseHB} {
		t.Run(fmt.Sprintf("interval=%s", interval), func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			node, sends := startLoneLeader(ctx, t, sparseOn, withRounds(interval))
			time.Sleep(testSparseHB)
			sends.reset()
			time.Sleep(60 * testSparseHB)

			leaderGap := sends.maxGap(Heartbeat, "trigger", "priceFix")
			heartbeatGap := sends.maxGap(Heartbeat)
			t.Logf("heartbeats %d, max leader gap %s, max heartbeat gap %s", sends.count(Heartbeat), leaderGap, heartbeatGap)
			// half the stickiness window plus a tick, with room for a loaded runner
			assert.Less(t, leaderGap, node.minElectionTimeout(), "leader gap reaches the stickiness window")
			assert.Less(t, heartbeatGap, node.maxHeartbeatGap()+2*testSparseHB, "heartbeats too far apart")
			if interval == testSparseHB {
				// rounds cover every tick: only the forced heartbeat goes out
				assert.Greater(t, heartbeatGap, node.maxHeartbeatGap()-2*testSparseHB)
			}
		})
	}
}

// In steady state a follower keeps hearing its leader within the stickiness
// window, so it refuses a stalled peer's vote, and never times the leader out,
// for short and long round intervals.
func TestSparseHeartbeatFollowerStaysSticky(t *testing.T) {
	for _, interval := range []time.Duration{0, testSparseHB, 4 * testSparseHB, 50 * testSparseHB} {
		t.Run(fmt.Sprintf("interval=%s", interval), func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			nodes := startCluster(ctx, t, 2, sparseOn, withRounds(interval))
			leader := waitForSingleLeader(t, nodes, "no initial leader")
			follower := otherNode(nodes, leader)
			term := leader.GetCurrentTerm()
			candidate := testPeerID(t)

			misses := 0
			deadline := time.Now().Add(2 * follower.maxElectionDetection())
			for time.Now().Before(deadline) {
				if !follower.hearsLeaderOtherThan(candidate) {
					misses++
				}
				require.Equal(t, Follower, follower.GetRole(), "follower timed out its leader")
				require.Equal(t, leader.GetHostId(), follower.GetLeader())
				time.Sleep(10 * time.Millisecond)
			}
			assert.Zero(t, misses, "follower stopped hearing its leader within the stickiness window")
			assert.Equal(t, term, leader.GetCurrentTerm())
			assert.Equal(t, term, follower.GetCurrentTerm())
		})
	}
}

// A follower that stalled and campaigned at a higher term can't depose the
// leader, and the leader's term reaches it within maxHeartbeatGap even when
// rounds make every other heartbeat redundant: it follows the leader again.
func TestSparseHeartbeatStalledFollowerFollowsWithinGap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// rounds every tick: only the forced heartbeat goes out
	nodes := startCluster(ctx, t, 2, sparseOn, withRounds(testSparseHB))
	leader := waitForSingleLeader(t, nodes, "no initial leader")
	follower := otherNode(nodes, leader)
	term := leader.GetCurrentTerm()

	// what its timer does after a stall: the leader messages it missed don't
	// count, so it campaigns
	follower.Mutex.Lock()
	follower.CooldownPeriod = 0
	follower.campaign(ctx, "timeout")
	follower.Mutex.Unlock()
	require.Equal(t, Candidate, follower.GetRole())
	require.Greater(t, follower.GetCurrentTerm(), term, "no election started")

	start := time.Now()
	for follower.GetRole() != Follower || follower.GetLeader() != leader.GetHostId() {
		require.Equal(t, Leader, leader.GetRole(), "leader deposed")
		require.Equal(t, term, leader.GetCurrentTerm(), "leader adopted the candidate's term")
		require.Less(t, time.Since(start), leader.maxHeartbeatGap()+3*testSparseHB, "candidate did not follow the leader within the heartbeat gap")
		time.Sleep(5 * time.Millisecond)
	}
	t.Logf("followed after %s", time.Since(start))

	deadline := time.Now().Add(10 * leader.minElectionTimeout())
	for time.Now().Before(deadline) {
		require.Equal(t, Leader, leader.GetRole(), "leader deposed")
		require.Equal(t, term, leader.GetCurrentTerm())
		require.Equal(t, Follower, follower.GetRole())
		time.Sleep(testLeaderInterval)
	}
}

// Failover is as fast as with every heartbeat: the leave path, and the
// election timer when the leader goes silent without leaving.
func TestSparseHeartbeatFailover(t *testing.T) {
	t.Run("disconnect", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		nodes := startCluster(ctx, t, 2, sparseOn, withRounds(4*testSparseHB), noSilenceFallback)
		leader := waitForSingleLeader(t, nodes, "no initial leader")
		survivor := otherNode(nodes, leader)

		time.Sleep(leader.CooldownPeriod) // past a first-election cooldown
		start := time.Now()
		require.NoError(t, leader.Host.Close())
		require.Eventually(t, func() bool { return survivor.GetRole() == Leader }, failoverBound, 10*time.Millisecond,
			"survivor did not take over at once")
		t.Logf("survivor led after %s", time.Since(start))
	})

	t.Run("timer", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		nodes := startCluster(ctx, t, 3, sparseOn, withRounds(4*testSparseHB))
		leader := waitForSingleLeader(t, nodes, "no initial leader")
		others := []*Raft{}
		for _, node := range nodes {
			if node != leader {
				others = append(others, node)
			}
		}
		time.Sleep(leader.CooldownPeriod)
		timeouts := electionsStarted(t, "timeout")

		// sendHeartbeat takes the mutex, so holding it silences the leader once
		// a heartbeat is due; the leader job ticker stops with the loop
		leader.Mutex.Lock()
		var unlock sync.Once
		release := func() { unlock.Do(leader.Mutex.Unlock) }
		defer release()

		start := time.Now()
		waitForSingleLeader(t, others, "no leader elected without the silent one")
		elapsed := time.Since(start)
		t.Logf("majority led after %s", elapsed)
		// the heartbeat due within maxHeartbeatGap blocks, then every missed
		// heartbeat and the campaign, then the stickiness window and votes
		assert.Less(t, elapsed, leader.maxHeartbeatGap()+leader.maxElectionDetection()+time.Second)
		assert.Greater(t, electionsStarted(t, "timeout"), timeouts)
		release()
	})
}

// The forced heartbeat must reach a follower without a leader before it can
// campaign: the earliest is after every missed heartbeat at the minimum
// election timeout.
func TestMaxHeartbeatGapBelowEarliestCampaign(t *testing.T) {
	for _, hb := range []time.Duration{testSparseHB, HEARTBEAT_TIMEOUT} {
		r := &Raft{HeartbeatTimeout: hb}
		assert.Less(t, r.maxHeartbeatGap(), (MaxMissedHeartbeats+1)*r.minElectionTimeout())
		assert.Less(t, r.sparseHeartbeatWindow()+hb, r.minElectionTimeout(), "leader gap reaches the stickiness window")
	}
}

// newSkippingLeader returns a sparse node whose stamps make the next tick skip.
func newSkippingLeader(ctx context.Context, t *testing.T) *Raft {
	t.Helper()
	node := newPublishTestNode(ctx, t)
	sparseOn(node)
	node.HandleCustomMessage = func(context.Context, Message) error { return nil }
	now := int64(time.Since(leaderClockEpoch))
	node.heartbeatSentAt.Store(now)
	node.leaderSentAt.Store(now)
	// 0 is recent in the test binary's first second
	node.rivalLeaderAt.Store(neverSent)
	require.True(t, node.skipHeartbeat())
	return node
}

// A leader that hears another peer's trigger or price fix stops skipping, so
// two leaders meet on heartbeats as fast as with the flag off. Its own, and
// messages any node sends, don't count.
func TestRivalLeaderMessageStopsSkipping(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for name, tc := range map[string]struct {
		msgType MessageType
		own     bool
		stops   bool
	}{
		"rival heartbeat": {Heartbeat, false, true},
		"own heartbeat":   {Heartbeat, true, false},
		"rival trigger":   {"trigger", false, true},
		"rival price fix": {"priceFix", false, true},
		"own trigger":     {"trigger", true, false},
		"price data":      {"priceData", false, false},
	} {
		t.Run(name, func(t *testing.T) {
			node := newSkippingLeader(ctx, t)
			author := testPeerID(t)
			if tc.own {
				author = node.GetHostId()
			}
			msg := leaderMessageFrom(author, tc.msgType)
			if tc.msgType == Heartbeat {
				msg = heartbeatFrom(t, author, 1)
			}
			require.NoError(t, node.handleFrom(ctx, msg, author))
			assert.Equal(t, !tc.stops, node.skipHeartbeat())
		})
	}
}

// A new term sends heartbeats until its own first one publishes: an earlier
// term's stamp doesn't let it skip, nor does that term's heartbeat finishing
// its publish late.
func TestNewTermDoesNotSkipOnEarlierHeartbeat(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node := newSkippingLeader(ctx, t)
	node.startElectionTimer()
	node.Mutex.Lock()
	node.setLeaderState()
	earlier := node.Resign
	node.setLeaderState()
	current := node.Resign
	node.Mutex.Unlock()
	assert.False(t, node.skipHeartbeat())

	node.stampHeartbeatSent(earlier, time.Now())
	assert.False(t, node.skipHeartbeat(), "an earlier term's heartbeat counted")

	node.stampHeartbeatSent(current, time.Now())
	assert.True(t, node.skipHeartbeat())
}
