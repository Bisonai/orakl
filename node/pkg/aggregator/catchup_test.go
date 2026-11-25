package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/raft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// published collects the messages of one type an aggregator publishes on its topic.
func published(t *testing.T, ctx context.Context, agg *Aggregator, msgType raft.MessageType) <-chan raft.Message {
	t.Helper()
	sub, err := agg.Raft.Topic.Subscribe()
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(ctx)
	out := make(chan raft.Message, 64)
	done := make(chan struct{})
	t.Cleanup(func() {
		cancel()
		sub.Cancel()
		<-done
	})
	go func() {
		defer close(done)
		for {
			raw, err := sub.Next(ctx)
			if err != nil {
				return
			}
			var msg raft.Message
			if json.Unmarshal(raw.Data, &msg) != nil || msg.Type != msgType {
				continue
			}
			select {
			case out <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// next returns the next message within wait, or false.
func next(ch <-chan raft.Message, wait time.Duration) (raft.Message, bool) {
	select {
	case msg := <-ch:
		return msg, true
	case <-time.After(wait):
		return raft.Message{}, false
	}
}

// messageRound decodes a price data or price fix message's RoundID.
func messageRound(t *testing.T, msg raft.Message) int32 {
	t.Helper()
	var m struct {
		RoundID int32 `json:"roundID"`
	}
	require.NoError(t, json.Unmarshal(msg.Data, &m))
	return m.RoundID
}

// messagePrice decodes a price data message's price.
func messagePrice(t *testing.T, msg raft.Message) int64 {
	t.Helper()
	var m PriceDataMessage
	require.NoError(t, decodeInner(msg.Data, &m))
	return m.PriceData
}

// A new leader behind its follower (fresh DB, or missed triggers as a
// follower) catches up to the follower's round instead of triggering rounds
// the follower already used and dedupes (#2867): its triggers move past the
// follower's locks within maxLaggingTriggers, the follower answers the next
// new round, and no round the follower already used gets a price or an
// aggregate. Leader start 30 is below the follower's lock window, 44 inside it.
func TestLaggingLeaderCatchesUpToFollower(t *testing.T) {
	const (
		followerRound      = 51
		maxLaggingTriggers = 1 // triggers at or below followerRound
	)
	for _, leaderStart := range []int32{30, 44} {
		t.Run(fmt.Sprint("leader at ", leaderStart), func(t *testing.T) {
			// published and messageRound decode JSON
			t.Setenv("P2P_ENCODING", "json")
			ctx := context.Background()

			leader := newRoundTestAggregator(t, ctx, fmt.Sprint("CATCHUP-LEADER-", leaderStart), nil)
			follower := newRoundTestAggregator(t, ctx, fmt.Sprint("CATCHUP-FOLLOWER-", leaderStart), nil)
			leaderID := leader.Raft.GetHostId()
			leader.Raft.Role = raft.Leader
			leader.Raft.LeaderID = leaderID
			leader.RoundID = leaderStart
			follower.Raft.LeaderID = leaderID
			follower.RoundID = followerRound
			seedRounds(follower, roundRange(42, followerRound)...)
			follower.LatestLocalAggregates.Store(follower.ID, &LocalAggregate{ConfigID: follower.ID, Value: 100, Timestamp: time.Now()})

			// Subscribe registers synchronously: no publish below is missed
			followerPrices := published(t, ctx, follower, PriceData)
			leaderFixes := published(t, ctx, leader, PriceFix)

			leaderRound := func() int32 {
				leader.mu.RLock()
				defer leader.mu.RUnlock()
				return leader.RoundID
			}

			var triggers []int32
			for len(triggers) < 15 {
				require.NoError(t, leader.LeaderJob(ctx))
				round := leaderRound()
				triggers = append(triggers, round)
				require.NoError(t, sendTrigger(t, ctx, follower, leaderID, round))

				msg, ok := next(followerPrices, 500*time.Millisecond)
				if !ok {
					continue // the follower deduped the round
				}
				require.Equal(t, round, messageRound(t, msg))
				if round >= 42 && round <= followerRound {
					assert.Equal(t, int64(-1), messagePrice(t, msg), "follower priced round %d it already used", round)
				}
				require.NoError(t, leader.HandlePriceDataMessage(ctx, msg))
				if round > followerRound {
					break
				}
			}

			lagging := 0
			for _, r := range triggers {
				if r <= followerRound {
					lagging++
				}
			}
			assert.LessOrEqual(t, lagging, maxLaggingTriggers, "triggers before catching up: %v", triggers)
			require.Greater(t, triggers[len(triggers)-1], int32(followerRound), "leader never caught up: %v", triggers)
			assert.Equal(t, int32(followerRound+1), triggers[len(triggers)-1], "first new round right after the follower's")

			// only the new round gets an aggregate, not the lagging one
			fix, ok := next(leaderFixes, time.Second)
			require.True(t, ok, "no price fix for the new round")
			assert.Equal(t, int32(followerRound+1), messageRound(t, fix))
			_, ok = next(leaderFixes, 200*time.Millisecond)
			assert.False(t, ok, "price fix for a round the follower already used")

			// the window stays bounded on both sides
			require.NoError(t, sendTrigger(t, ctx, leader, leaderID, followerRound+1))
			assert.Equal(t, roundRange(followerRound+1-(roundHistory-1), followerRound+1), sortedKeys(follower.roundTriggers.locked))
			assert.Equal(t, roundRange(followerRound+1, followerRound+1), sortedKeys(leader.roundTriggers.locked))
		})
	}
}

func sendPriceData(t *testing.T, ctx context.Context, agg *Aggregator, from string, round, latest int32) {
	t.Helper()
	data, err := encodeInner(PriceDataMessage{RoundID: round, PriceData: 100, Timestamp: time.Now(), LatestRoundID: latest})
	require.NoError(t, err)
	require.NoError(t, agg.HandlePriceDataMessage(ctx, raft.Message{Type: PriceData, SentFrom: from, Data: data}))
}

// Once caught up, the leader drops every price for a round a peer already
// used, also its own and an older node's (no LatestRoundID).
func TestCaughtUpLeaderDropsAllPricesForStaleRounds(t *testing.T) {
	t.Setenv("P2P_ENCODING", "json")
	ctx := context.Background()
	leader := newRoundTestAggregator(t, ctx, "CATCHUP-STALE", nil)
	self := leader.Raft.GetHostId()
	leader.Raft.Role = raft.Leader
	leader.Raft.LeaderID = self
	leader.RoundID = 32 // triggered 31 and 32 before any reply

	sendPriceData(t, ctx, leader, "ahead-peer", 31, 51)
	sendPriceData(t, ctx, leader, self, 32, 32)
	sendPriceData(t, ctx, leader, "old-peer", 32, 0)

	leader.mu.RLock()
	assert.Equal(t, int32(51), leader.RoundID)
	leader.mu.RUnlock()
	leader.roundPrices.mu.Lock()
	defer leader.roundPrices.mu.Unlock()
	assert.Equal(t, []int32{31, 32}, sortedKeys(leader.roundPrices.locked))
	assert.Empty(t, leader.roundPrices.prices, "no price stored for a stale round")
}

// A round whose collection timeout started before the leader caught up gets
// no aggregate when the timeout fires.
func TestCollectionTimeoutSkipsRoundStaleSinceStart(t *testing.T) {
	t.Setenv("P2P_ENCODING", "json")
	ctx := context.Background()
	leader := newRoundTestAggregator(t, ctx, "CATCHUP-TIMEOUT", nil)
	self := leader.Raft.GetHostId()
	leader.Raft.Role = raft.Leader
	leader.Raft.LeaderID = self
	leader.RoundID = 32
	fixes := published(t, ctx, leader, PriceFix)

	// own price for 31 stored, its timeout pending; then a peer's reply for
	// 32 catches the leader up past 31
	leader.roundPrices.prices[31] = []int64{100}
	leader.roundPrices.senders[31] = []string{self}
	sendPriceData(t, ctx, leader, "ahead-peer", 32, 51)

	leader.startPriceCollectionTimeout(ctx, 31, time.Now())

	leader.roundPrices.mu.Lock()
	assert.True(t, leader.roundPrices.locked[31])
	leader.roundPrices.mu.Unlock()
	_, ok := next(fixes, 200*time.Millisecond)
	assert.False(t, ok, "price fix for a round a peer already used")
}

// triggerMsg builds a trigger as the leader publishes it.
func triggerMsg(t *testing.T, from string, round int32, ts time.Time) raft.Message {
	t.Helper()
	data, err := encodeInner(TriggerMessage{LeaderID: from, RoundID: round, Timestamp: ts})
	require.NoError(t, err)
	return raft.Message{Type: Trigger, SentFrom: from, Data: data}
}

// messageLatestRound decodes a price data message's LatestRoundID.
func messageLatestRound(t *testing.T, msg raft.Message) int32 {
	t.Helper()
	var m PriceDataMessage
	require.NoError(t, decodeInner(msg.Data, &m))
	return m.LatestRoundID
}

// A leader on a fresh or lost round state that triggers the round its
// follower is at (used with another trigger) gets no price for it but moves
// past it, and its next round is a new one (#2927).
func TestLeaderReusingFollowersCurrentRoundCatchesUp(t *testing.T) {
	const followerRound = 51
	t.Setenv("P2P_ENCODING", "json")
	ctx := context.Background()

	leader := newRoundTestAggregator(t, ctx, "REUSE-LEADER", nil)
	follower := newRoundTestAggregator(t, ctx, "REUSE-FOLLOWER", nil)
	leaderID := leader.Raft.GetHostId()
	leader.Raft.Role = raft.Leader
	leader.Raft.LeaderID = leaderID
	leader.RoundID = followerRound - 1
	follower.Raft.LeaderID = leaderID
	follower.RoundID = followerRound
	seedRounds(follower, roundRange(42, followerRound)...) // used with the previous leader
	follower.LatestLocalAggregates.Store(follower.ID, &LocalAggregate{ConfigID: follower.ID, Value: 100, Timestamp: time.Now()})

	followerPrices := published(t, ctx, follower, PriceData)
	leaderFixes := published(t, ctx, leader, PriceFix)

	leaderRound := func() int32 {
		leader.mu.RLock()
		defer leader.mu.RUnlock()
		return leader.RoundID
	}

	require.NoError(t, leader.LeaderJob(ctx))
	require.Equal(t, int32(followerRound), leaderRound())
	require.NoError(t, sendTrigger(t, ctx, follower, leaderID, followerRound))

	msg, ok := next(followerPrices, 500*time.Millisecond)
	require.True(t, ok, "follower ignored a reused trigger for its current round")
	assert.Equal(t, int32(followerRound), messageRound(t, msg))
	assert.Equal(t, int64(-1), messagePrice(t, msg), "follower priced a round it already used")
	assert.Equal(t, int32(followerRound+1), messageLatestRound(t, msg))
	require.NoError(t, leader.HandlePriceDataMessage(ctx, msg))

	// the next trigger is a round neither node used
	require.NoError(t, leader.LeaderJob(ctx))
	newRound := leaderRound()
	require.Greater(t, newRound, int32(followerRound+1))
	require.NoError(t, sendTrigger(t, ctx, follower, leaderID, newRound))
	msg, ok = next(followerPrices, 500*time.Millisecond)
	require.True(t, ok, "no price for the new round")
	assert.Equal(t, newRound, messageRound(t, msg))
	assert.Equal(t, int64(100), messagePrice(t, msg))
	require.NoError(t, leader.HandlePriceDataMessage(ctx, msg))

	fix, ok := next(leaderFixes, time.Second)
	require.True(t, ok, "no price fix for the new round")
	assert.Equal(t, newRound, messageRound(t, fix))
	_, ok = next(leaderFixes, 200*time.Millisecond)
	assert.False(t, ok, "price fix for the reused round")
}

// A duplicate delivery of the trigger for the round in progress is not a
// reused round: no second answer, and the round and the follower's RoundID
// stay as they are.
func TestDuplicateTriggerForCurrentRoundIgnored(t *testing.T) {
	t.Setenv("P2P_ENCODING", "json")
	ctx := context.Background()

	follower := newRoundTestAggregator(t, ctx, "REUSE-DUPLICATE", nil)
	follower.Raft.LeaderID = "leader-peer"
	follower.RoundID = 50
	follower.LatestLocalAggregates.Store(follower.ID, &LocalAggregate{ConfigID: follower.ID, Value: 100, Timestamp: time.Now()})
	followerPrices := published(t, ctx, follower, PriceData)

	trigger := triggerMsg(t, "leader-peer", 51, time.Now())
	require.NoError(t, follower.HandleTriggerMessage(ctx, trigger))
	msg, ok := next(followerPrices, 500*time.Millisecond)
	require.True(t, ok, "no price for the round")
	assert.Equal(t, int32(51), messageRound(t, msg))
	assert.Equal(t, int64(100), messagePrice(t, msg))
	assert.Equal(t, int32(51), messageLatestRound(t, msg))

	require.NoError(t, follower.HandleTriggerMessage(ctx, trigger))
	_, ok = next(followerPrices, 200*time.Millisecond)
	assert.False(t, ok, "answer to a duplicate trigger")
	follower.mu.RLock()
	assert.Equal(t, int32(51), follower.RoundID)
	follower.mu.RUnlock()
}
