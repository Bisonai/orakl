package raft

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A leader message handled before the new leader's first heartbeat waits for
// that heartbeat instead of being dropped (#2999); one from a peer that never
// becomes the leader is rejected after the minimum election timeout.
func TestAwaitLeader(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("heartbeat after the wait started", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		leader := testPeerID(t)
		done := make(chan bool, 1)
		go func() { done <- node.AwaitLeader(ctx, leader) }()
		require.Eventually(t, func() bool {
			node.Mutex.Lock()
			defer node.Mutex.Unlock()
			return node.leaderFollowed != nil
		}, time.Second, time.Millisecond, "AwaitLeader not waiting")

		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 1)))
		select {
		case ok := <-done:
			assert.True(t, ok)
		case <-time.After(node.minElectionTimeout() / 2):
			t.Fatal("AwaitLeader still waiting after the heartbeat")
		}
	})

	t.Run("already the leader", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		leader := testPeerID(t)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 1)))
		assert.True(t, node.AwaitLeader(ctx, leader))
	})

	t.Run("never the leader", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		start := time.Now()
		assert.False(t, node.AwaitLeader(ctx, testPeerID(t)))
		assert.GreaterOrEqual(t, time.Since(start), node.minElectionTimeout())
	})

	t.Run("another leader heard", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, testPeerID(t), 1)))
		start := time.Now()
		assert.False(t, node.AwaitLeader(ctx, testPeerID(t)))
		assert.Less(t, time.Since(start), node.minElectionTimeout(), "waited although another leader is heard")
	})

	t.Run("becomes leader itself", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		done := make(chan bool, 1)
		go func() { done <- node.AwaitLeader(ctx, testPeerID(t)) }()
		require.Eventually(t, func() bool {
			node.Mutex.Lock()
			defer node.Mutex.Unlock()
			return node.leaderFollowed != nil
		}, time.Second, time.Millisecond, "AwaitLeader not waiting")
		node.Mutex.Lock()
		node.setLeaderState()
		node.Mutex.Unlock()
		select {
		case ok := <-done:
			assert.False(t, ok)
		case <-time.After(node.minElectionTimeout() / 2):
			t.Fatal("AwaitLeader still waiting after this node took the lead")
		}
	})

	t.Run("ctx done", func(t *testing.T) {
		node := newSilenceTestNode(ctx, t)
		stopped, stop := context.WithCancel(ctx)
		stop()
		assert.False(t, node.AwaitLeader(stopped, testPeerID(t)))
	})
}
