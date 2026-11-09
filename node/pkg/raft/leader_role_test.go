package raft

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

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
