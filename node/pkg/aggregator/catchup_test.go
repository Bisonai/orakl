package aggregator

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/raft"
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
			out <- msg
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

// A new leader behind its follower (fresh DB or missed triggers) catches up to
// the follower's round instead of triggering rounds the follower already used
// and dedupes (#2867): its triggers move past the follower's locks within
// maxLaggingTriggers, the follower answers the next new round, and no round
// the follower already used gets an aggregate.
func TestLaggingLeaderCatchesUpToFollower(t *testing.T) {
	const (
		leaderStart        = 30
		followerRound      = 51
		maxLaggingTriggers = 1 // triggers at or below followerRound
	)
	ctx := context.Background()

	leader := newRoundTestAggregator(t, ctx, "CATCHUP-LEADER", nil)
	follower := newRoundTestAggregator(t, ctx, "CATCHUP-FOLLOWER", nil)
	leaderID := leader.Raft.GetHostId()
	leader.Raft.Role = raft.Leader
	leader.Raft.LeaderID = leaderID
	leader.RoundID = leaderStart
	follower.Raft.LeaderID = leaderID
	follower.RoundID = followerRound
	seedRounds(follower, roundRange(42, followerRound)...)
	follower.LatestLocalAggregates.Store(follower.ID, &LocalAggregate{ConfigID: follower.ID, Value: 100})

	followerPrices := published(t, ctx, follower, PriceData)
	leaderFixes := published(t, ctx, leader, PriceFix)
	// let the subscriptions settle before the first publish
	time.Sleep(50 * time.Millisecond)

	var triggers []int32
	for len(triggers) < 15 {
		require.NoError(t, leader.LeaderJob(ctx))
		round := leader.RoundID
		triggers = append(triggers, round)
		require.NoError(t, sendTrigger(t, ctx, follower, leaderID, round))

		msg, ok := next(followerPrices, 500*time.Millisecond)
		if !ok {
			continue // the follower deduped the round
		}
		require.Equal(t, round, messageRound(t, msg))
		require.NoError(t, leader.HandlePriceDataMessage(ctx, msg))
		if round > followerRound {
			break
		}
	}

	lagging := 0
	for _, r := range triggers {
		assert.False(t, r >= 42 && r <= followerRound, "leader triggered round %d the follower already used", r)
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
}
