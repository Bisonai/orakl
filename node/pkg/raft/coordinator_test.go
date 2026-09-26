//nolint:all
package raft

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// newFollower builds a bare follower Raft with no host/pubsub — enough to exercise
// applyHeartbeat's follower paths, which never touch the network.
func newFollower(term int) *Raft {
	return &Raft{
		Role:             Follower,
		Term:             term,
		Mutex:            sync.Mutex{},
		HeartbeatTimeout: HEARTBEAT_TIMEOUT,
	}
}

// TestCoordinatorFanout proves a combined heartbeat is split per feed: only the
// feeds present in the message are updated, and each gets the sender as leader and
// the carried term, with its missed-heartbeat counter reset — identical to a
// per-feed heartbeat today. Feeds absent from the message are left untouched.
func TestCoordinatorFanout(t *testing.T) {
	feed1 := newFollower(0)
	feed1.MissedHeartbeats = 2
	feed2 := newFollower(0)
	feed3 := newFollower(0) // not addressed by the leader

	c := &HeartbeatCoordinator{
		feeds: map[int32]*Raft{1: feed1, 2: feed2, 3: feed3},
	}

	c.fanout("leader-peer", map[int32]int{1: 5, 2: 7})

	assert.Equal(t, 5, feed1.GetCurrentTerm(), "feed1 term follows the batch")
	assert.Equal(t, "leader-peer", feed1.GetLeader())
	assert.Equal(t, 0, feed1.MissedHeartbeats, "missed heartbeats reset")

	assert.Equal(t, 7, feed2.GetCurrentTerm())
	assert.Equal(t, "leader-peer", feed2.GetLeader())

	assert.Equal(t, 0, feed3.GetCurrentTerm(), "unaddressed feed untouched")
	assert.Equal(t, "", feed3.GetLeader())
}

// TestCoordinatorFanoutUnknownFeed proves a term for a feed this node does not
// run is silently ignored (no panic, no effect on other feeds).
func TestCoordinatorFanoutUnknownFeed(t *testing.T) {
	feed1 := newFollower(0)
	c := &HeartbeatCoordinator{feeds: map[int32]*Raft{1: feed1}}

	c.fanout("leader-peer", map[int32]int{1: 3, 999: 42})

	assert.Equal(t, 3, feed1.GetCurrentTerm())
}

func TestHeartbeatBatchEnabledFlag(t *testing.T) {
	t.Setenv("P2P_HEARTBEAT_BATCH", "")
	assert.False(t, HeartbeatBatchEnabled(), "default off")

	for _, v := range []string{"1", "true", "on", "yes", "TRUE"} {
		t.Setenv("P2P_HEARTBEAT_BATCH", v)
		assert.True(t, HeartbeatBatchEnabled(), v)
	}

	t.Setenv("P2P_HEARTBEAT_BATCH", "off")
	assert.False(t, HeartbeatBatchEnabled())
}

func TestControlTopicName(t *testing.T) {
	os.Unsetenv("P2P_HEARTBEAT_CONTROL_TOPIC")
	assert.Equal(t, DefaultControlTopic, ControlTopicName())

	t.Setenv("P2P_HEARTBEAT_CONTROL_TOPIC", "custom-control")
	assert.Equal(t, "custom-control", ControlTopicName())
}
