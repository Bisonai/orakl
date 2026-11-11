package raft

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failoverBound is the failover these tests expect once the leader leaves: the
// leave grace and a lone election, with room for a loaded runner. Detecting it
// through missed heartbeats alone takes up to 3 s (maxElectionDetection).
const failoverBound = 1500 * time.Millisecond

// runningNode is a cluster node whose Run can be stopped and started again.
type runningNode struct {
	*Raft
	cancel context.CancelFunc
	done   chan struct{}
}

func (n *runningNode) start(ctx context.Context) {
	runCtx, cancel := context.WithCancel(ctx)
	n.cancel = cancel
	n.done = make(chan struct{})
	go func(done chan struct{}) {
		defer close(done)
		n.Run(runCtx)
	}(n.done)
}

func (n *runningNode) stop() {
	n.cancel()
	<-n.done
}

// startStoppableCluster is startCluster with each node's Run stoppable.
func startStoppableCluster(ctx context.Context, t *testing.T, n int, configure ...func(*Raft)) ([]*Raft, map[*Raft]*runningNode) {
	t.Helper()
	nodes := newCluster(ctx, t, n, configure...)
	running := make(map[*Raft]*runningNode, n)
	for _, node := range nodes {
		r := &runningNode{Raft: node}
		r.start(ctx)
		running[node] = r
	}
	t.Cleanup(func() {
		for _, r := range running {
			r.stop()
		}
	})
	waitForSubscribed(t, nodes, n)
	return nodes, running
}

// noSilenceFallback rules out the silent-peer fallback as what elects.
func noSilenceFallback(r *Raft) { r.PeerSilenceTimeout = time.Hour }

// A leader whose connection drops is replaced at once: the survivor counts it
// out as a voter and wins on its own vote, without the silent-peer fallback.
func TestLeaderDisconnectElectsAtOnce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := startCluster(ctx, t, 2, noSilenceFallback)
	leader := waitForSingleLeader(t, nodes, "no initial leader")
	survivor := otherNode(nodes, leader)
	before := electionsStarted(t, "disconnect")

	time.Sleep(leader.CooldownPeriod) // past a first-election cooldown
	start := time.Now()
	require.NoError(t, leader.Host.Close())
	require.Eventually(t, func() bool { return survivor.GetRole() == Leader }, failoverBound, 10*time.Millisecond,
		"survivor did not take over at once")
	t.Logf("survivor led after %s", time.Since(start))
	assert.Equal(t, before+1, electionsStarted(t, "disconnect"))
}

// A leader that stops (its aggregator stopped, or the node shuts down) leaves
// the topic while connected: the follower elects at once, reason shutdown.
func TestLeaderStopElectsAtOnce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes, running := startStoppableCluster(ctx, t, 2, noSilenceFallback)
	leader := waitForSingleLeader(t, nodes, "no initial leader")
	survivor := otherNode(nodes, leader)
	before := electionsStarted(t, "shutdown")

	time.Sleep(leader.CooldownPeriod) // past a first-election cooldown
	start := time.Now()
	running[leader].stop()
	assert.Equal(t, Follower, leader.GetRole(), "stopped leader did not resign")
	require.Eventually(t, func() bool { return survivor.GetRole() == Leader }, failoverBound, 10*time.Millisecond,
		"survivor did not take over at once")
	t.Logf("survivor led after %s", time.Since(start))
	assert.Equal(t, before+1, electionsStarted(t, "shutdown"))

	// the stopped node follows the survivor once it runs again
	running[leader].start(ctx)
	waitForSingleLeader(t, nodes, "restarted node did not follow")
	assert.Equal(t, survivor.GetHostId(), leader.GetLeader())
}

// reconnect drops the connection between a and b and dials again at once.
func reconnect(ctx context.Context, t *testing.T, a, b *Raft) {
	t.Helper()
	require.NoError(t, a.Host.Network().ClosePeer(b.Host.ID()))
	require.NoError(t, a.Host.Connect(ctx, peer.AddrInfo{ID: b.Host.ID(), Addrs: b.Host.Addrs()}))
}

func leaveElections(t *testing.T) float64 {
	return electionsStarted(t, "disconnect") + electionsStarted(t, "shutdown")
}

// A connection restored within the grace, as the reconnect worker does, elects
// nothing, however often it flaps.
func TestQuickReconnectDoesNotElect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := startCluster(ctx, t, 2)
	leader := waitForSingleLeader(t, nodes, "no initial leader")
	follower := otherNode(nodes, leader)
	term := follower.GetCurrentTerm()
	before := leaveElections(t)

	for i := 0; i < 5; i++ {
		reconnect(ctx, t, follower, leader)
		time.Sleep(follower.peerLeaveGrace() / 2)
	}
	time.Sleep(2 * follower.peerLeaveGrace())

	assert.Equal(t, before, leaveElections(t), "a quick reconnect started an election")
	got, ok := singleLeader(nodes)
	require.True(t, ok, "no single leader")
	assert.Equal(t, leader, got, "leader changed")
	assert.Equal(t, term, follower.GetCurrentTerm())
}

// A connection back only after the grace hands over once: the old leader finds
// its successor at a higher term and follows it, with no election back.
func TestLateReconnectHandsOverOnce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := startCluster(ctx, t, 2)
	leader := waitForSingleLeader(t, nodes, "no initial leader")
	follower := otherNode(nodes, leader)

	time.Sleep(leader.CooldownPeriod) // past a first-election cooldown

	require.NoError(t, follower.Host.Network().ClosePeer(leader.Host.ID()))
	require.Eventually(t, func() bool { return follower.GetRole() == Leader }, failoverBound, 10*time.Millisecond,
		"follower did not take over")
	require.NoError(t, follower.Host.Connect(ctx, peer.AddrInfo{ID: leader.Host.ID(), Addrs: leader.Host.Addrs()}))

	require.Eventually(t, func() bool {
		got, ok := singleLeader(nodes)
		return ok && got == follower
	}, testElectionWait, 10*time.Millisecond, "old leader did not follow its successor")
	term := follower.GetCurrentTerm()
	before := electionsStarted(t, "timeout") + leaveElections(t)

	time.Sleep(follower.maxElectionDetection() + follower.CooldownPeriod)
	got, ok := singleLeader(nodes)
	require.True(t, ok, "no single leader")
	assert.Equal(t, follower, got, "leadership moved again")
	assert.Equal(t, term, follower.GetCurrentTerm())
	assert.Equal(t, before, electionsStarted(t, "timeout")+leaveElections(t), "elected again after the reconnect")
}

// A node without leave handling (PeerLeaveGrace < 0, as before this change)
// beside one with it: wire messages are the same, so whichever leads, stopping
// it hands over and the restarted node follows.
func TestMixedFleetFailover(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	legacy := true
	nodes, running := startStoppableCluster(ctx, t, 2, func(r *Raft) {
		if legacy {
			r.PeerLeaveGrace = -1
			legacy = false
		}
	})
	for round := 0; round < 2; round++ {
		leader := waitForSingleLeader(t, nodes, "no leader")
		survivor := otherNode(nodes, leader)
		running[leader].stop()
		waitForSingleLeader(t, []*Raft{survivor}, "survivor did not take over")
		running[leader].start(ctx)
		require.Equal(t, survivor, waitForSingleLeader(t, nodes, "restarted node did not follow"))
	}
}

// With 3 nodes both followers see the leader leave; the jittered grace keeps
// them from campaigning at once and splitting the vote.
func TestLeaderStopThreeNodesElectsOnce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes, running := startStoppableCluster(ctx, t, 3, noSilenceFallback)
	leader := waitForSingleLeader(t, nodes, "no initial leader")
	survivors := []*Raft{}
	for _, node := range nodes {
		if node != leader {
			survivors = append(survivors, node)
		}
	}

	// a survivor that campaigned in the first election keeps the cooldown
	time.Sleep(leader.CooldownPeriod)

	start := time.Now()
	running[leader].stop()
	waitForSingleLeader(t, survivors, "survivors did not elect")
	t.Logf("survivors led after %s", time.Since(start))
	assert.Less(t, time.Since(start), 2*failoverBound, "split vote: fell back to the timeout")
}
