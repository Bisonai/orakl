//nolint:all
package raft

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		feeds: map[string]*Raft{"feed1": feed1, "feed2": feed2, "feed3": feed3},
	}

	c.fanout("leader-peer", map[string]int{"feed1": 5, "feed2": 7})

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
	c := &HeartbeatCoordinator{feeds: map[string]*Raft{"feed1": feed1}}

	c.fanout("leader-peer", map[string]int{"feed1": 3, "unknown-feed": 42})

	assert.Equal(t, 3, feed1.GetCurrentTerm())
}

func TestHeartbeatBatchEnabledFlag(t *testing.T) {
	t.Setenv("P2P_HEARTBEAT_BATCH", "")
	assert.False(t, HeartbeatBatchEnabled(), "default off")

	for _, v := range []string{"1", "true", "TRUE", "True", "t"} {
		t.Setenv("P2P_HEARTBEAT_BATCH", v)
		assert.True(t, HeartbeatBatchEnabled(), v)
	}

	// strconv.ParseBool (codebase convention) only accepts the standard bool
	// literals; on/yes/off and junk are all off.
	for _, v := range []string{"off", "0", "on", "yes", "garbage"} {
		t.Setenv("P2P_HEARTBEAT_BATCH", v)
		assert.False(t, HeartbeatBatchEnabled(), v)
	}
}

func TestControlTopicName(t *testing.T) {
	os.Unsetenv("P2P_HEARTBEAT_CONTROL_TOPIC")
	assert.Equal(t, DefaultControlTopic, ControlTopicName())

	t.Setenv("P2P_HEARTBEAT_CONTROL_TOPIC", "custom-control")
	assert.Equal(t, "custom-control", ControlTopicName())
}

// batchFrom encodes a combined heartbeat from sender as a received pubsub message.
func batchFrom(t *testing.T, sender string, terms map[string]int) *pubsub.Message {
	t.Helper()
	inner, err := encodeInner(BatchHeartbeatMessage{Terms: terms})
	require.NoError(t, err)
	data, err := encodeMessage(Message{Type: BatchHeartbeat, SentFrom: sender, Data: inner, Timestamp: time.Now()})
	require.NoError(t, err)
	id, err := peer.Decode(sender)
	require.NoError(t, err)
	return &pubsub.Message{Message: &pubsubpb.Message{Data: data, From: []byte(id)}}
}

func newLeader(term int) *Raft {
	r := &Raft{Role: Leader, Term: term, HeartbeatTimeout: HEARTBEAT_TIMEOUT}
	r.leaderLoops.Store(1)
	return r
}

// TestStalledPublishDoesNotBlockReceive proves a blocked Publish neither stops the
// receive loop draining the buffer nor queues ticks (#2570).
func TestStalledPublishDoesNotBlockReceive(t *testing.T) {
	t.Setenv("P2P_HEARTBEAT_BATCH", "true")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	node := newPublishTestNode(ctx, t)
	const bufferSize = 8
	c := NewHeartbeatCoordinator(node.Host, node.Topic, bufferSize)

	release := make(chan struct{})
	var publishes atomic.Int32
	c.publish = func(ctx context.Context, _ []byte) error {
		publishes.Add(1)
		select {
		case <-release:
		case <-ctx.Done():
		}
		return nil
	}

	followed := newFollower(0)
	followed.MissedHeartbeats = 2
	c.Reset(map[string]*Raft{"led": newLeader(1), "followed": followed})

	go c.Run(ctx)
	require.Eventually(t, func() bool { return publishes.Load() == 1 }, 5*time.Second, 5*time.Millisecond, "publish entered")

	// more messages than the buffer holds: this only completes if Run keeps draining
	const total = bufferSize * 4
	sender := testPeerID(t)
	for term := 1; term <= total; term++ {
		select {
		case c.buffer <- batchFrom(t, sender, map[string]int{"followed": term}):
		case <-time.After(5 * time.Second):
			t.Fatal("receive buffer not drained while publish is stalled")
		}
	}
	require.Eventually(t, func() bool { return followed.GetCurrentTerm() == total }, 5*time.Second, 5*time.Millisecond, "fan-out continues")
	followed.Mutex.Lock()
	missed := followed.MissedHeartbeats
	followed.Mutex.Unlock()
	assert.Equal(t, 0, missed)

	time.Sleep(6 * HEARTBEAT_TIMEOUT)
	assert.EqualValues(t, 1, publishes.Load(), "no further publish while one is stalled")

	close(release)
	require.Eventually(t, func() bool { return publishes.Load() >= 2 }, 5*time.Second, 5*time.Millisecond, "broadcast resumes")
	time.Sleep(HEARTBEAT_TIMEOUT / 2)
	assert.Less(t, publishes.Load(), int32(5), "stalled ticks are skipped, not replayed")
}

func TestBroadcastFlagOffDoesNotPublish(t *testing.T) {
	t.Setenv("P2P_HEARTBEAT_BATCH", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node := newPublishTestNode(ctx, t)
	c := NewHeartbeatCoordinator(node.Host, node.Topic, 8)
	var publishes atomic.Int32
	c.publish = func(context.Context, []byte) error {
		publishes.Add(1)
		return nil
	}
	c.Reset(map[string]*Raft{"led": newLeader(1)})

	go c.Run(ctx)
	time.Sleep(3 * HEARTBEAT_TIMEOUT)
	assert.Zero(t, publishes.Load())
}

// TestBroadcastCountsPublished proves combined heartbeats appear in the p2p
// publish counters under their own type, with the exact payload size.
func TestBroadcastCountsPublished(t *testing.T) {
	for _, encoding := range []string{"json", "msgpack"} {
		t.Run(encoding, func(t *testing.T) {
			t.Setenv("P2P_ENCODING", encoding)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			node := newPublishTestNode(ctx, t)
			sub, err := node.Topic.Subscribe()
			require.NoError(t, err)
			defer sub.Cancel()

			c := NewHeartbeatCoordinator(node.Host, node.Topic, 8)
			c.Reset(map[string]*Raft{"led": newLeader(7), "followed": newFollower(7)})

			messages := counterValue(t, metricnames.P2PPublishedMessages, BatchHeartbeat)
			size := counterValue(t, metricnames.P2PPublishedBytes, BatchHeartbeat)

			c.broadcast(ctx)

			published, err := sub.Next(ctx)
			require.NoError(t, err)
			msg, err := decodeMessage(published.Data)
			require.NoError(t, err)
			var batch BatchHeartbeatMessage
			require.NoError(t, decodeInner(msg.Data, &batch))
			assert.Equal(t, map[string]int{"led": 7}, batch.Terms, "only led feeds")

			assert.Equal(t, messages+1, counterValue(t, metricnames.P2PPublishedMessages, BatchHeartbeat))
			assert.Equal(t, size+float64(len(published.Data)), counterValue(t, metricnames.P2PPublishedBytes, BatchHeartbeat))
		})
	}
}

func TestBroadcastPublishFailureNotCounted(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node := newPublishTestNode(ctx, t)
	c := NewHeartbeatCoordinator(node.Host, node.Topic, 8)
	c.Reset(map[string]*Raft{"led": newLeader(1)})

	// a success first, so the series exist and a wrong name can't pass as 0 == 0
	c.broadcast(ctx)
	messages := counterValue(t, metricnames.P2PPublishedMessages, BatchHeartbeat)
	size := counterValue(t, metricnames.P2PPublishedBytes, BatchHeartbeat)
	require.Positive(t, messages)

	c.publish = func(context.Context, []byte) error { return errors.New("publish failed") }
	c.broadcast(ctx)

	assert.Equal(t, messages, counterValue(t, metricnames.P2PPublishedMessages, BatchHeartbeat))
	assert.Equal(t, size, counterValue(t, metricnames.P2PPublishedBytes, BatchHeartbeat))
}

// TestBatchedHeartbeatMatchesPerFeed drives the same heartbeat through the per-feed
// handler and the combined fan-out and requires identical resulting state, stale
// terms included.
func TestBatchedHeartbeatMatchesPerFeed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	node := newPublishTestNode(ctx, t)
	leaderID := testPeerID(t)

	for _, term := range []int{4, 5, 6} {
		t.Run(fmt.Sprintf("term=%d", term), func(t *testing.T) {
			newState := func() *Raft {
				r := newFollower(5)
				r.Host = node.Host
				r.LeaderID = "previous"
				r.MissedHeartbeats = 2
				return r
			}

			perFeed := newState()
			inner, err := encodeInner(HeartbeatMessage{LeaderID: leaderID, Term: term})
			require.NoError(t, err)
			require.NoError(t, perFeed.handleHeartbeat(Message{Type: Heartbeat, SentFrom: leaderID, Data: inner}))

			batched := newState()
			c := &HeartbeatCoordinator{host: node.Host, feeds: map[string]*Raft{"feed": batched}}
			c.handleRaw(batchFrom(t, leaderID, map[string]int{"feed": term}))

			assert.Equal(t, 0, batched.MissedHeartbeats, "missed heartbeats reset")
			assert.NotNil(t, batched.ElectionTimer, "election timer restarted")
			assert.Equal(t, perFeed.MissedHeartbeats, batched.MissedHeartbeats)
			assert.Equal(t, perFeed.Term, batched.Term)
			assert.Equal(t, perFeed.Role, batched.Role)
			assert.Equal(t, perFeed.LeaderID, batched.LeaderID)
			assert.Equal(t, perFeed.ElectionTimer != nil, batched.ElectionTimer != nil)
		})
	}
}

// TestBatchHeartbeatForgedSenderDropped proves a message whose self-declared sender
// differs from the signed From has no effect on any feed.
func TestBatchHeartbeatForgedSenderDropped(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	node := newPublishTestNode(ctx, t)

	feed := newFollower(5)
	c := &HeartbeatCoordinator{host: node.Host, feeds: map[string]*Raft{"feed": feed}}

	forged := batchFrom(t, testPeerID(t), map[string]int{"feed": 1000})
	other, err := peer.Decode(testPeerID(t))
	require.NoError(t, err)
	forged.From = []byte(other)

	c.handleRaw(forged)

	assert.Equal(t, 5, feed.GetCurrentTerm())
	assert.Equal(t, "", feed.GetLeader())
}

// TestLeaderSnapshotRequiresLiveLoop proves a group stopped by ctx cancel, which
// keeps Role == Leader, is no longer advertised in combined heartbeats.
func TestLeaderSnapshotRequiresLiveLoop(t *testing.T) {
	testCtx, testCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer testCancel()
	node := newPublishTestNode(testCtx, t)
	node.LeaderJob = func(context.Context) error { return nil }

	_, isLeader := node.leaderTermSnapshot()
	assert.False(t, isLeader, "follower")

	ctx, cancel := context.WithCancel(testCtx)
	defer cancel()
	node.Mutex.Lock()
	node.becomeLeader(ctx)
	node.Mutex.Unlock()

	term, isLeader := node.leaderTermSnapshot()
	assert.True(t, isLeader, "running leader")
	assert.Equal(t, 1, term)

	cancel()
	require.Eventually(t, func() bool {
		_, isLeader := node.leaderTermSnapshot()
		return !isLeader
	}, 5*time.Second, 5*time.Millisecond, "stopped leader not advertised")
	node.Mutex.Lock()
	role := node.Role
	node.Mutex.Unlock()
	assert.Equal(t, Leader, role, "stop leaves the role as is, matching master")
}

// TestFanoutBeforeRunIsSafe drives fan-out concurrently with Raft.Run starting;
// under -race this fails if the election timer is created in Run.
func TestFanoutBeforeRunIsSafe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	node := newPublishTestNode(ctx, t)
	c := &HeartbeatCoordinator{host: node.Host, feeds: map[string]*Raft{"feed": node}}

	done := make(chan struct{})
	go func() {
		defer close(done)
		c.fanout("leader-peer", map[string]int{"feed": 3})
	}()
	runCtx, stop := context.WithCancel(ctx)
	defer stop()
	go node.Run(runCtx)
	<-done

	assert.Equal(t, 3, node.GetCurrentTerm())
}

// TestResetClearsReplacedRaft proves a Raft replaced under the same name stops
// pointing at the coordinator, so it can't suppress its per-feed heartbeat.
func TestResetClearsReplacedRaft(t *testing.T) {
	c := &HeartbeatCoordinator{}
	old, replacement := newFollower(0), newFollower(0)

	c.Reset(map[string]*Raft{"feed": old})
	assert.Same(t, c, old.hbCoordinator.Load())

	c.Reset(map[string]*Raft{"feed": replacement})
	assert.Nil(t, old.hbCoordinator.Load())
	assert.Same(t, c, replacement.hbCoordinator.Load())
}

// TestBatchHeartbeatWithinGossipSubLimit bounds a combined heartbeat covering 233
// feeds against the GossipSub message size limit, in both wire formats.
func TestBatchHeartbeatWithinGossipSubLimit(t *testing.T) {
	terms := make(map[string]int, 233)
	for i := 0; i < 233; i++ {
		terms[fmt.Sprintf("T%03d-USDT", i)] = 12345
	}
	sender := testPeerID(t)

	for _, encoding := range []string{"json", "msgpack"} {
		t.Run(encoding, func(t *testing.T) {
			t.Setenv("P2P_ENCODING", encoding)
			raw := batchFrom(t, sender, terms)
			t.Logf("%d feeds, %s: %d bytes", len(terms), encoding, len(raw.Data))
			assert.Less(t, len(raw.Data), pubsub.DefaultMaxMessageSize)
		})
	}
}
