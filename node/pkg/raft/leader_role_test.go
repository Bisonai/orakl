package raft

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countHeartbeats counts the heartbeats node publishes, as peers would see them.
func countHeartbeats(ctx context.Context, t *testing.T, node *Raft) *atomic.Int64 {
	t.Helper()
	sub, err := node.Topic.Subscribe()
	require.NoError(t, err)
	t.Cleanup(sub.Cancel)

	count := &atomic.Int64{}
	go func() {
		for {
			raw, err := sub.Next(ctx)
			if err != nil {
				return
			}
			msg, err := node.unmarshalMessage(raw.Data)
			if err == nil && msg.Type == Heartbeat {
				count.Add(1)
			}
		}
	}()
	return count
}

func TestResignedLoopParkedOnFirstHeartbeatDoesNothing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, jobs := newLeaderLoopNode(ctx, t)
	heartbeats := countHeartbeats(ctx, t, node)

	// the loop's first heartbeat parks on the mutex held here
	node.Mutex.Lock()
	node.becomeLeader(ctx)
	time.Sleep(testLoopSettle)
	node.ResignLeader()
	node.Term += 5 // as if adopted from the new leader
	node.Mutex.Unlock()

	time.Sleep(testLoopSettle)
	assert.Zero(t, heartbeats.Load(), "heartbeat after resign")
	assert.Zero(t, jobs.Load(), "leader job after resign")
}

func TestResignedLoopParkedOnTickHeartbeatDoesNothing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, jobs := newLeaderLoopNode(ctx, t)
	heartbeats := countHeartbeats(ctx, t, node)

	node.Mutex.Lock()
	node.becomeLeader(ctx)
	node.Mutex.Unlock()
	require.Eventually(t, func() bool { return jobs.Load() >= 2 && heartbeats.Load() >= 2 }, 5*time.Second, testLeaderInterval, "loop not running")

	// holding the mutex past a tick parks the loop in its heartbeat branch; the
	// wait also lets heartbeats already published arrive
	node.Mutex.Lock()
	time.Sleep(testLoopSettle)
	before := heartbeats.Load()
	node.ResignLeader()
	node.Mutex.Unlock()

	time.Sleep(testLoopSettle)
	assert.Equal(t, before, heartbeats.Load())
}

// A check on the role alone would pass here, since the node leads again.
func TestResignedLoopStaysSilentAfterReelection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, jobs := newLeaderLoopNode(ctx, t)
	heartbeats := countHeartbeats(ctx, t, node)
	// no ticks: each loop can only send its first heartbeat and job
	node.HeartbeatTimeout = time.Hour
	node.LeaderJobTimeout = time.Hour

	node.Mutex.Lock()
	node.becomeLeader(ctx)
	time.Sleep(testLoopSettle)
	node.ResignLeader()
	node.becomeLeader(ctx)
	node.Mutex.Unlock()

	require.Eventually(t, func() bool { return heartbeats.Load() >= 1 && jobs.Load() >= 1 }, 5*time.Second, testLeaderInterval, "new term's loop not running")
	time.Sleep(testLoopSettle)
	assert.Equal(t, int64(1), heartbeats.Load())
	assert.Equal(t, int64(1), jobs.Load())
}

func grantedVoteFor(t *testing.T, node *Raft) Message {
	t.Helper()
	data, err := encodeInner(ReplyRequestVoteMessage{VoteGranted: true, LeaderID: node.GetHostId()})
	require.NoError(t, err)
	return Message{Type: ReplyVote, SentFrom: testPeerID(t), Data: data}
}

// newCandidate returns a node that just started an election and has jobs counted.
func newCandidate(ctx context.Context, t *testing.T) (*Raft, *atomic.Int64) {
	t.Helper()
	node, jobs := newLeaderLoopNode(ctx, t)
	node.CooldownPeriod = 0 // so only the role can stop a later election
	node.MissedHeartbeats = MaxMissedHeartbeats
	node.startElection(ctx)
	require.Equal(t, Candidate, node.GetRole())
	return node, jobs
}

func TestLeaderDoesNotStartElection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, _ := newCandidate(ctx, t)
	require.NoError(t, node.handleMessage(ctx, grantedVoteFor(t, node)))
	require.Equal(t, Leader, node.GetRole())

	term := node.GetCurrentTerm()
	started := electionsStarted(t, metrics.RaftElectionReasonTimeout)

	node.Mutex.Lock()
	node.MissedHeartbeats = MaxMissedHeartbeats
	node.Mutex.Unlock()
	node.startElection(ctx)

	assert.Equal(t, Leader, node.GetRole())
	assert.Equal(t, term, node.GetCurrentTerm())
	assert.Equal(t, started, electionsStarted(t, metrics.RaftElectionReasonTimeout))

	node.Mutex.Lock()
	defer node.Mutex.Unlock()
	assert.False(t, node.ElectionTimer.Stop(), "timer armed on a leader")
}

// The timer firing during the winning vote must not leave a loop running on a
// node that is no longer leader.
func TestLeaderLoopStopsAfterTimerFiresOnNewLeader(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, jobs := newCandidate(ctx, t)
	require.NoError(t, node.handleMessage(ctx, grantedVoteFor(t, node)))
	require.Equal(t, Leader, node.GetRole())
	term := node.GetCurrentTerm()

	// enough expiries to get past the missed heartbeat count
	for i := 0; i <= MaxMissedHeartbeats; i++ {
		node.startElection(ctx)
	}
	assert.Equal(t, Leader, node.GetRole())

	require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, testPeerID(t), term+2)))

	assert.Equal(t, Follower, node.GetRole())
	assert.True(t, node.ElectionTimer.Stop(), "timer not restarted on resign")
	assertLoopStopped(t, jobs)
}

func TestHeartbeatStepsCandidateDown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for name, ahead := range map[string]int{"same term": 0, "higher term": 1} {
		t.Run(name, func(t *testing.T) {
			node, jobs := newCandidate(ctx, t)
			term := node.GetCurrentTerm() + ahead
			leader := testPeerID(t)
			node.ElectionTimer.Stop() // to see the heartbeat restart it

			require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, term)))

			assert.Equal(t, Follower, node.GetRole())
			assert.Equal(t, leader, node.GetLeader())
			assert.Equal(t, term, node.GetCurrentTerm())
			assert.True(t, node.ElectionTimer.Stop(), "timer not restarted")

			require.NoError(t, node.handleMessage(ctx, grantedVoteFor(t, node)))

			assert.Equal(t, Follower, node.GetRole(), "late vote")
			assert.Equal(t, leader, node.GetLeader())
			assert.Equal(t, term, node.GetCurrentTerm())
			time.Sleep(testLoopSettle)
			assert.Zero(t, jobs.Load(), "late vote")
		})
	}
}

// A candidate whose RequestVote a live leader ignored (#2760) follows that
// leader's lower-term heartbeats, keeping its own term.
func TestLowerTermHeartbeatEndsCandidacyWithoutLiveLeader(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, jobs := newCandidate(ctx, t)
	node.Term = 5
	leader := testPeerID(t)

	require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 4)))

	assert.Equal(t, Follower, node.GetRole())
	assert.Equal(t, leader, node.GetLeader())
	assert.Equal(t, 5, node.GetCurrentTerm())

	require.NoError(t, node.handleMessage(ctx, grantedVoteFor(t, node)))
	assert.Equal(t, Follower, node.GetRole(), "late vote")
	time.Sleep(testLoopSettle)
	assert.Zero(t, jobs.Load(), "late vote")
}

// A stale heartbeat, e.g. from a deposed leader, doesn't move a node off a
// leader it still hears (open question in #2747).
func TestLowerTermHeartbeatIgnoredWhileLeaderHeard(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, _ := newLeaderLoopNode(ctx, t)
	leader := testPeerID(t)
	require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 5)))

	require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, testPeerID(t), 4)))

	assert.Equal(t, Follower, node.GetRole())
	assert.Equal(t, leader, node.GetLeader())
	assert.Equal(t, 5, node.GetCurrentTerm())
}
