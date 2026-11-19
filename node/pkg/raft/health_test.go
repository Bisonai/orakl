package raft

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withHealth gives each node a CanLead it controls through the returned map.
func withHealth(health map[*Raft]*atomic.Bool) func(*Raft) {
	return func(r *Raft) {
		ok := &atomic.Bool{}
		ok.Store(true)
		health[r] = ok
		r.CanLead = ok.Load
	}
}

// A leader whose reporter can't submit hands the feed to its peer within a
// bound, and doesn't win it back while unhealthy (#2914).
func TestUnhealthyLeaderHandsOver(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	health := map[*Raft]*atomic.Bool{}
	nodes := startCluster(ctx, t, 2, withHealth(health))
	leader := waitForSingleLeader(t, nodes, "no initial leader")
	peer := otherNode(nodes, leader)

	health[leader].Store(false)
	start := time.Now()
	require.Eventually(t, func() bool { return leader.GetRole() != Leader }, time.Second, 10*time.Millisecond, "unhealthy leader did not resign")
	newLeader := waitForSingleLeader(t, nodes, "peer did not take over")
	t.Logf("peer led after %s", time.Since(start))
	assert.Equal(t, peer, newLeader, "unhealthy node won the feed back")

	// it stays with the peer past the unhealthy node's hold-off
	time.Sleep(2 * leader.maxElectionDetection())
	assert.Equal(t, Leader, peer.GetRole())
	assert.Equal(t, Follower, leader.GetRole())
}

// An unhealthy node with a live peer leaves an election to it for the
// hold-off, then campaigns, so the feed still gets a leader if no peer can.
func TestUnhealthyNodeLeavesElectionToPeer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nodes := newCluster(ctx, t, 2)
	node, peer := nodes[0], nodes[1]
	node.CanLead = func() bool { return false }
	sub, err := peer.Topic.Subscribe()
	require.NoError(t, err)
	defer sub.Cancel()
	require.Eventually(t, func() bool { return node.SubscribersCount() == 1 }, testElectionWait, 10*time.Millisecond)

	forceNextElection(node)
	node.startElection(ctx)
	assert.Equal(t, Follower, node.GetRole(), "campaigned against a live peer while unhealthy")
	assert.Equal(t, 0, node.GetCurrentTerm())

	// the hold-off is over
	node.Mutex.Lock()
	node.unhealthyCampaignAfter = time.Now().Add(-time.Millisecond)
	node.Mutex.Unlock()
	forceNextElection(node)
	node.startElection(ctx)
	assert.Equal(t, Candidate, node.GetRole())
	assert.Equal(t, 1, node.GetCurrentTerm())
}

// With no peer to take over, an unhealthy leader keeps the feed: resigning
// would only stop its rounds.
func TestLoneUnhealthyLeaderKeepsLeading(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, _ := newLeaderLoopNode(ctx, t)
	node.CanLead = func() bool { return false }

	node.Mutex.Lock()
	node.becomeLeader(ctx)
	term := node.Term
	node.Mutex.Unlock()

	time.Sleep(testLoopSettle)
	assert.Equal(t, Leader, node.GetRole())
	assert.Equal(t, term, node.GetCurrentTerm())

	// and a lone unhealthy node campaigns at once
	node.Mutex.Lock()
	node.ResignLeader()
	node.Mutex.Unlock()
	forceNextElection(node)
	node.startElection(ctx)
	assert.Equal(t, Candidate, node.GetRole())
}

// When every node is unhealthy the feed settles on one leader instead of
// moving around: each resigns at most once per HealthResignBackoff.
func TestAllUnhealthySettles(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	health := map[*Raft]*atomic.Bool{}
	nodes := startCluster(ctx, t, 2, withHealth(health), func(r *Raft) {
		r.HeartbeatTimeout = testSparseHB
		r.CooldownPeriod = 10 * testSparseHB
	})
	waitForSingleLeader(t, nodes, "no initial leader")
	for _, ok := range health {
		ok.Store(false)
	}

	// each hands over once, then the one that won back keeps it
	var leader *Raft
	require.Eventually(t, func() bool {
		var ok bool
		leader, ok = singleLeader(nodes)
		return ok && leader.healthResignedAt.Load() != 0
	}, testElectionWait, 50*time.Millisecond, "no leader that already resigned for health")

	term := leader.GetCurrentTerm()
	time.Sleep(3 * leader.maxElectionDetection())
	assert.Equal(t, Leader, leader.GetRole(), "feed moved again within the backoff")
	assert.Equal(t, term, leader.GetCurrentTerm())

	// healthy again: the next unhealthy spell hands over at once
	health[leader].Store(true)
	require.Eventually(t, func() bool { return leader.healthResignedAt.Load() == 0 }, time.Second, 10*time.Millisecond)
}
