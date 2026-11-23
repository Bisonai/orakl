package aggregator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// triggerPrice sends this node a trigger as if from its leader and returns
// the price it answers with.
func triggerPrice(t *testing.T, aggregate *LocalAggregate) int64 {
	t.Helper()
	// published and messageRound decode JSON
	t.Setenv("P2P_ENCODING", "json")
	ctx := context.Background()
	agg := newRoundTestAggregator(t, ctx, "LOCAL-AGE-"+t.Name(), nil)
	agg.Raft.LeaderID = "leader-peer"
	if aggregate != nil {
		aggregate.ConfigID = agg.ID
		agg.LatestLocalAggregates.Store(agg.ID, aggregate)
	}
	prices := published(t, ctx, agg, PriceData)
	require.NoError(t, sendTrigger(t, ctx, agg, "leader-peer", 2))
	msg, ok := next(prices, 2*time.Second)
	require.True(t, ok, "no price data message")
	require.Equal(t, int32(2), messageRound(t, msg))
	return messagePrice(t, msg)
}

func TestTriggerVotesFreshLocalAggregate(t *testing.T) {
	assert.Equal(t, int64(100), triggerPrice(t, &LocalAggregate{Value: 100, Timestamp: time.Now()}))
}

// The fetcher stops publishing a config whose sources are stale; its last
// local aggregate must not be voted on (#2964).
func TestTriggerVotesNoPriceForStaleLocalAggregate(t *testing.T) {
	// e.g. AUCTION-KRW: the last local aggregate is a minute old
	assert.Equal(t, int64(-1), triggerPrice(t, &LocalAggregate{Value: 100, Timestamp: time.Now().Add(-time.Minute)}))
}

func TestTriggerVotesLocalAggregateWithinMaxAge(t *testing.T) {
	t.Setenv("LOCAL_AGGREGATE_MAX_AGE", "2m") // read by NewLatestLocalAggregates
	assert.Equal(t, int64(100), triggerPrice(t, &LocalAggregate{Value: 100, Timestamp: time.Now().Add(-time.Minute)}))
}

func TestTriggerVotesNoPriceWithoutLocalAggregate(t *testing.T) {
	assert.Equal(t, int64(-1), triggerPrice(t, nil))
}

// Each bus message is stored on its own goroutine, so an older local
// aggregate can arrive after a newer one.
func TestStoreKeepsNewerLocalAggregate(t *testing.T) {
	aggregates := NewLatestLocalAggregates()
	now := time.Now()
	aggregates.Store(1, &LocalAggregate{ConfigID: 1, Value: 200, Timestamp: now})
	aggregates.Store(1, &LocalAggregate{ConfigID: 1, Value: 100, Timestamp: now.Add(-time.Second)})

	got, ok := aggregates.Load(1)
	require.True(t, ok)
	assert.Equal(t, int64(200), got.Value)

	aggregates.Store(1, &LocalAggregate{ConfigID: 1, Value: 300, Timestamp: now.Add(time.Second)})
	got, ok = aggregates.Load(1)
	require.True(t, ok)
	assert.Equal(t, int64(300), got.Value)
}

// A config removed by a sync drops its cached local aggregate.
func TestRetainDropsRemovedConfigs(t *testing.T) {
	aggregates := NewLatestLocalAggregates()
	aggregates.Store(1, &LocalAggregate{ConfigID: 1, Value: 100, Timestamp: time.Now()})
	aggregates.Store(2, &LocalAggregate{ConfigID: 2, Value: 200, Timestamp: time.Now()})

	aggregates.Retain([]Config{{ID: 2}})
	_, ok := aggregates.Load(1)
	assert.False(t, ok)
	_, ok = aggregates.Load(2)
	assert.True(t, ok)
}
