package raft

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func leaderMessageFrom(leaderID string, msgType MessageType) Message {
	return Message{Type: msgType, SentFrom: leaderID, Data: []byte(`{}`)}
}

func newLivenessTestNode(ctx context.Context, t *testing.T) *Raft {
	t.Helper()
	node := newSilenceTestNode(ctx, t)
	node.LeaderMessageTypes = map[MessageType]bool{"trigger": true, "priceFix": true}
	node.CooldownPeriod = 0
	return node
}

// expireTimer runs what the election timer does when it fires.
func expireTimer(ctx context.Context, node *Raft) {
	node.startElection(ctx)
}

// A trigger or price fix from the leader resets the election timer like a
// heartbeat. Messages any node sends don't.
func TestLeaderMessagesHoldOffElection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for name, tc := range map[string]struct {
		msgType MessageType
		holds   bool
	}{
		"trigger":   {"trigger", true},
		"priceFix":  {"priceFix", true},
		"priceData": {"priceData", false},
		"proof":     {"proof", false},
	} {
		t.Run(name, func(t *testing.T) {
			node := newLivenessTestNode(ctx, t)
			leader := testPeerID(t)
			require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 1)))

			for i := 0; i < 3*(MaxMissedHeartbeats+1); i++ {
				require.NoError(t, node.handleMessage(ctx, leaderMessageFrom(leader, tc.msgType)))
				expireTimer(ctx, node)
			}
			term, _, role := voteState(node)
			if tc.holds {
				assert.Equal(t, Follower, role)
				assert.Equal(t, 1, term)
				assert.Equal(t, leader, node.GetLeader())
			} else {
				assert.NotEqual(t, Follower, role, "no election without leader messages")
				assert.Greater(t, term, 1)
			}
		})
	}
}

// Without leader messages the timer runs out as before.
func TestElectionAfterLeaderMessagesStop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node := newLivenessTestNode(ctx, t)
	leader := testPeerID(t)
	require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 1)))
	require.NoError(t, node.handleMessage(ctx, leaderMessageFrom(leader, "trigger")))
	expireTimer(ctx, node)
	require.Equal(t, Follower, node.GetRole())

	before := electionsStarted(t, "timeout")
	for i := 0; i < MaxMissedHeartbeats+1; i++ {
		expireTimer(ctx, node)
	}
	assert.Equal(t, Candidate, node.GetRole())
	assert.Equal(t, 2, node.GetCurrentTerm())
	assert.Equal(t, before+1, electionsStarted(t, "timeout"))
}

// Triggers carry no term. A deposed leader's late triggers don't count once the
// follower follows its successor, so they can't keep it attached to either: it
// times out the silent successor and campaigns.
func TestDeposedLeaderTriggersDoNotHoldOffElection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node := newLivenessTestNode(ctx, t)
	deposed, successor := testPeerID(t), testPeerID(t)
	require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, deposed, 1)))
	require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, successor, 2)))
	require.Equal(t, successor, node.GetLeader())

	for i := 0; i < MaxMissedHeartbeats+1; i++ {
		require.NoError(t, node.handleMessage(ctx, leaderMessageFrom(deposed, "trigger")))
		require.NoError(t, node.handleMessage(ctx, leaderMessageFrom(deposed, "priceFix")))
		assert.NotEqual(t, deposed, node.GetLeader(), "a trigger moved the follower back to the deposed leader")
		expireTimer(ctx, node)
	}
	assert.Equal(t, Candidate, node.GetRole())
	assert.Equal(t, 3, node.GetCurrentTerm())
}

// Liveness keys on the signed author: a message naming the leader in SentFrom
// but published by another peer doesn't count (#2782).
func TestForgedLeaderMessageDoesNotHoldOffElection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node := newLivenessTestNode(ctx, t)
	leader, forger := testPeerID(t), testPeerID(t)
	require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 1)))

	for i := 0; i < MaxMissedHeartbeats+1; i++ {
		require.NoError(t, node.handleFrom(ctx, leaderMessageFrom(leader, "trigger"), forger))
		expireTimer(ctx, node)
	}
	assert.Equal(t, Candidate, node.GetRole())
}

// End to end on the real timer: a follower whose leader sends only triggers
// keeps following it, and campaigns soon after they stop.
func TestTriggersKeepFollowerWithoutHeartbeats(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node := newPublishTestNode(ctx, t)
	node.HandleCustomMessage = func(context.Context, Message) error { return nil }
	node.LeaderJob = func(context.Context) error { return nil }
	node.LeaderMessageTypes = map[MessageType]bool{"trigger": true}
	done := make(chan struct{})
	go func() {
		defer close(done)
		node.Run(ctx)
	}()
	defer func() { cancel(); <-done }()

	leader := testPeerID(t)
	require.Eventually(t, func() bool {
		return node.handleMessage(ctx, heartbeatFrom(t, leader, 1)) == nil && node.GetLeader() == leader
	}, time.Second, 10*time.Millisecond)

	// longer than the slowest timeout: every missed heartbeat at the maximum
	// election timeout, twice
	deadline := time.Now().Add(2 * node.maxElectionDetection())
	for time.Now().Before(deadline) {
		require.NoError(t, node.handleMessage(ctx, leaderMessageFrom(leader, "trigger")))
		require.Equal(t, Follower, node.GetRole(), "timed out a leader that sends triggers")
		require.Equal(t, leader, node.GetLeader())
		time.Sleep(node.HeartbeatTimeout)
	}

	// the lone node wins on its own vote once the triggers stop
	require.Eventually(t, func() bool { return node.GetRole() == Leader }, testElectionWait, 50*time.Millisecond)
}
