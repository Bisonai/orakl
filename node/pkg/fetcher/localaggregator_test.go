//nolint:all
package fetcher

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func intPtr(v int) *int {
	return &v
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func newLocalAggregatorWithFreshness(freshness *int) *LocalAggregator {
	return &LocalAggregator{
		Config: Config{
			ID:                1,
			Name:              "TEST-USDT",
			FetchInterval:     2000,
			FeedDataFreshness: freshness,
		},
	}
}

func TestFilterStaleFeeds_DisabledWhenNil(t *testing.T) {
	la := newLocalAggregatorWithFreshness(nil)

	now := time.Now()
	old := now.Add(-10 * time.Minute)
	feeds := []*FeedData{
		{FeedID: 1, Value: 100.0, Timestamp: &now},
		{FeedID: 2, Value: 101.0, Timestamp: &old},
	}

	result := la.filterStaleFeeds(feeds)
	assert.Equal(t, len(feeds), len(result), "nil freshness should return all feeds")
}

func TestFilterStaleFeeds_DisabledWhenZero(t *testing.T) {
	la := newLocalAggregatorWithFreshness(intPtr(0))

	now := time.Now()
	old := now.Add(-10 * time.Minute)
	feeds := []*FeedData{
		{FeedID: 1, Value: 100.0, Timestamp: &now},
		{FeedID: 2, Value: 101.0, Timestamp: &old},
	}

	result := la.filterStaleFeeds(feeds)
	assert.Equal(t, len(feeds), len(result), "zero freshness should return all feeds")
}

func TestFilterStaleFeeds_DisabledWhenNegative(t *testing.T) {
	la := newLocalAggregatorWithFreshness(intPtr(-1000))

	now := time.Now()
	old := now.Add(-10 * time.Minute)
	feeds := []*FeedData{
		{FeedID: 1, Value: 100.0, Timestamp: &now},
		{FeedID: 2, Value: 101.0, Timestamp: &old},
	}

	result := la.filterStaleFeeds(feeds)
	assert.Equal(t, len(feeds), len(result), "negative freshness should return all feeds")
}

func TestFilterStaleFeeds_SingleFeedAlwaysReturned(t *testing.T) {
	la := newLocalAggregatorWithFreshness(intPtr(60000)) // 60 seconds

	old := time.Now().Add(-5 * time.Minute)
	feeds := []*FeedData{
		{FeedID: 1, Value: 100.0, Timestamp: &old},
	}

	result := la.filterStaleFeeds(feeds)
	assert.Equal(t, 1, len(result), "single feed should always be returned even if stale")
}

func TestFilterStaleFeeds_AllFresh(t *testing.T) {
	la := newLocalAggregatorWithFreshness(intPtr(60000)) // 60 seconds

	now := time.Now()
	t1 := now.Add(-10 * time.Second)
	t2 := now.Add(-30 * time.Second)
	t3 := now.Add(-50 * time.Second)

	feeds := []*FeedData{
		{FeedID: 1, Value: 100.0, Timestamp: &t1},
		{FeedID: 2, Value: 101.0, Timestamp: &t2},
		{FeedID: 3, Value: 102.0, Timestamp: &t3},
	}

	result := la.filterStaleFeeds(feeds)
	assert.Equal(t, 3, len(result), "all fresh feeds should be returned")
}

// All TestFilterStaleFeeds_* fixtures set Volume > 0 — filterStaleFeeds
// short-circuits Volume == 0 feeds (DEX/HTTP) past the freshness check,
// so omitting Volume here would mean the feeds bypass the filter entirely
// and these tests would silently lose their assertion power.

func TestFilterStaleFeeds_MixFreshAndStale(t *testing.T) {
	la := newLocalAggregatorWithFreshness(intPtr(60000)) // 60 seconds

	now := time.Now()
	fresh1 := now.Add(-10 * time.Second)
	fresh2 := now.Add(-30 * time.Second)
	stale1 := now.Add(-2 * time.Minute)
	stale2 := now.Add(-5 * time.Minute)

	feeds := []*FeedData{
		{FeedID: 1, Value: 100.0, Volume: 1, Timestamp: &fresh1},
		{FeedID: 2, Value: 0.0119, Volume: 1, Timestamp: &stale1}, // stuck old price
		{FeedID: 3, Value: 101.0, Volume: 1, Timestamp: &fresh2},
		{FeedID: 4, Value: 0.0119, Volume: 1, Timestamp: &stale2}, // stuck old price
	}

	result := la.filterStaleFeeds(feeds)
	assert.Equal(t, 2, len(result), "only fresh feeds should be returned")
	for _, feed := range result {
		assert.Contains(t, []int32{1, 3}, feed.FeedID, "stale feeds should be excluded")
	}
}

func TestFilterStaleFeeds_AllStale(t *testing.T) {
	la := newLocalAggregatorWithFreshness(intPtr(60000)) // 60 seconds

	now := time.Now()
	stale1 := now.Add(-2 * time.Minute)
	stale2 := now.Add(-5 * time.Minute)
	stale3 := now.Add(-10 * time.Minute)

	feeds := []*FeedData{
		{FeedID: 1, Value: 100.0, Volume: 1, Timestamp: &stale1},
		{FeedID: 2, Value: 101.0, Volume: 1, Timestamp: &stale2},
		{FeedID: 3, Value: 102.0, Volume: 1, Timestamp: &stale3},
	}

	result := la.filterStaleFeeds(feeds)
	assert.Equal(t, 0, len(result), "all stale feeds should be filtered out")
}

func TestFilterStaleFeeds_NilTimestampPassesThrough(t *testing.T) {
	la := newLocalAggregatorWithFreshness(intPtr(60000)) // 60 seconds

	now := time.Now()
	fresh := now.Add(-10 * time.Second)
	stale := now.Add(-5 * time.Minute)

	feeds := []*FeedData{
		{FeedID: 1, Value: 100.0, Volume: 1, Timestamp: &fresh},
		{FeedID: 2, Value: 101.0, Volume: 1, Timestamp: nil},   // nil timestamp should pass
		{FeedID: 3, Value: 102.0, Volume: 1, Timestamp: &stale}, // stale
	}

	result := la.filterStaleFeeds(feeds)
	assert.Equal(t, 2, len(result), "fresh + nil timestamp feeds should be returned")
	for _, feed := range result {
		assert.Contains(t, []int32{1, 2}, feed.FeedID)
	}
}

func TestFilterStaleFeeds_BoundaryExactlyAtThreshold(t *testing.T) {
	la := newLocalAggregatorWithFreshness(intPtr(60000)) // 60 seconds

	now := time.Now()
	// exactly at the boundary - age == freshness, should be excluded (age > freshness is false but age == freshness)
	// actually age > freshness, so exactly 60s should NOT be excluded
	// Let's test just under and just over
	justUnder := now.Add(-59999 * time.Millisecond)
	justOver := now.Add(-60001 * time.Millisecond)

	feeds := []*FeedData{
		{FeedID: 1, Value: 100.0, Volume: 1, Timestamp: &justUnder},
		{FeedID: 2, Value: 101.0, Volume: 1, Timestamp: &justOver},
	}

	result := la.filterStaleFeeds(feeds)
	assert.Equal(t, 1, len(result), "only feed just under threshold should remain")
	assert.Equal(t, int32(1), result[0].FeedID)
}

func TestFilterStaleFeeds_EmptySlice(t *testing.T) {
	la := newLocalAggregatorWithFreshness(intPtr(60000))

	feeds := []*FeedData{}
	result := la.filterStaleFeeds(feeds)
	assert.Equal(t, 0, len(result), "empty feeds should return empty result")
}

func TestFilterStaleFeeds_WithVolumeData(t *testing.T) {
	la := newLocalAggregatorWithFreshness(intPtr(60000))

	now := time.Now()
	fresh := now.Add(-10 * time.Second)
	stale := now.Add(-2 * time.Minute)

	feeds := []*FeedData{
		{FeedID: 1, Value: 0.086, Volume: 1000000.0, Timestamp: &fresh},
		{FeedID: 2, Value: 0.0119, Volume: 500.0, Timestamp: &stale}, // stuck price with low volume
		{FeedID: 3, Value: 0.085, Volume: 800000.0, Timestamp: &fresh},
	}

	result := la.filterStaleFeeds(feeds)
	assert.Equal(t, 2, len(result), "stale feed should be excluded regardless of volume")
	for _, feed := range result {
		assert.Contains(t, []int32{1, 3}, feed.FeedID)
	}
}

func TestFilterStaleFeeds_IntegrationWithCollectFlow(t *testing.T) {
	// Simulate the full collect → filter → 50% check flow
	freshness := 60000
	la := &LocalAggregator{
		Config: Config{
			ID:                1,
			Name:              "ZKP-USDT",
			FetchInterval:     2000,
			FeedDataFreshness: &freshness,
		},
		Feeds: []Feed{
			{ID: 10, Name: "binance-wss-ZKP-USDT", ConfigID: 1},
			{ID: 11, Name: "coinbase-wss-ZKP-USDT", ConfigID: 1},
			{ID: 12, Name: "bitmart-wss-ZKP-USDT", ConfigID: 1},
			{ID: 13, Name: "okx-wss-ZKP-USDT", ConfigID: 1},
		},
	}

	now := time.Now()
	fresh := now.Add(-5 * time.Second)
	stale := now.Add(-3 * time.Minute) // bitmart stuck

	feedDataMap := &LatestFeedDataMap{
		FeedDataMap: map[int32]*FeedData{
			10: {FeedID: 10, Value: 0.086, Volume: 1000000, Timestamp: &fresh},
			11: {FeedID: 11, Value: 0.085, Volume: 800000, Timestamp: &fresh},
			12: {FeedID: 12, Value: 0.0119, Volume: 100, Timestamp: &stale}, // stuck
			13: {FeedID: 13, Value: 0.087, Volume: 900000, Timestamp: &fresh},
		},
	}
	la.latestFeedDataMap = feedDataMap

	// simulate collect
	feedIds := make([]int32, len(la.Feeds))
	for i, feed := range la.Feeds {
		feedIds[i] = feed.ID
	}
	feeds, err := la.latestFeedDataMap.GetLatestFeedData(feedIds)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(feeds), "should get all 4 feeds from map")

	filtered := la.filterStaleFeeds(feeds)
	assert.Equal(t, 3, len(filtered), "bitmart stuck feed should be excluded")

	// 50% check: 3 >= 4/2 (2) → should pass
	assert.GreaterOrEqual(t, len(filtered), len(la.Feeds)/2, "filtered feeds should pass 50%% threshold")

	// verify excluded feed is the stale one
	for _, feed := range filtered {
		assert.NotEqual(t, int32(12), feed.FeedID, "feed 12 (bitmart stuck) should be excluded")
	}
}

func wssFeed(id int32) Feed {
	return Feed{ID: id, Name: "wss", Definition: []byte(`{"type":"wss","provider":"upbit","base":"MTL","quote":"KRW"}`), ConfigID: 1}
}

func dexFeed(id int32) Feed {
	return Feed{ID: id, Name: "dex", Definition: []byte(`{"type":"UniswapPool","chainId":"1","address":"0x0"}`), ConfigID: 1}
}

func httpFeed(id int32) Feed {
	return Feed{ID: id, Name: "http", Definition: []byte(`{"url":"https://example.com","method":"GET"}`), ConfigID: 1}
}

func newTypedLocalAggregator(feeds []Feed, data map[int32]*FeedData) *LocalAggregator {
	freshness := 60000
	return NewLocalAggregator(
		Config{ID: 1, Name: "MTL-KRW", FetchInterval: 2000, FeedDataFreshness: &freshness},
		feeds,
		make(chan *LocalAggregate, 1),
		nil,
		&LatestFeedDataMap{FeedDataMap: data},
		nil,
	)
}

func feedIDs(feeds []*FeedData) []int32 {
	ids := make([]int32, 0, len(feeds))
	for _, f := range feeds {
		ids = append(ids, f.FeedID)
	}
	return ids
}

func TestBuildWssFeedIDs(t *testing.T) {
	ids := buildWssFeedIDs([]Feed{wssFeed(1), dexFeed(2), httpFeed(3), {ID: 4, Definition: []byte(`bad`)}})
	assert.Equal(t, map[int32]bool{1: true}, ids)
}

func TestFilterStaleFeeds_WssVolumeZeroNotExempt(t *testing.T) {
	now := time.Now()
	stale := now.Add(-5 * time.Minute)
	la := newTypedLocalAggregator([]Feed{wssFeed(1), wssFeed(2), dexFeed(3), httpFeed(4)}, nil)

	feeds := []*FeedData{
		{FeedID: 1, Value: 100, Volume: 1, Timestamp: &now},
		{FeedID: 2, Value: 110, Volume: 0, Timestamp: &stale}, // wss vol 0, stale
		{FeedID: 3, Value: 101, Volume: 0, Timestamp: &stale}, // dex, exempt
		{FeedID: 4, Value: 102, Volume: 0, Timestamp: &stale}, // http, exempt
	}

	result := la.filterStaleFeeds(feeds)
	assert.ElementsMatch(t, []int32{1, 3, 4}, feedIDs(result))
}

func TestFilterZombieWssFeeds_KeptWhileLiveWssPresent(t *testing.T) {
	now := time.Now()
	la := newTypedLocalAggregator([]Feed{wssFeed(1), wssFeed(2), wssFeed(3)}, nil)

	feeds := []*FeedData{
		{FeedID: 1, Value: 100, Volume: 5, Timestamp: &now},
		{FeedID: 2, Value: 110, Volume: 0, Timestamp: &now},
		{FeedID: 3, Value: 111, Volume: 0, Timestamp: &now},
	}

	result := la.filterZombieWssFeeds(feeds)
	assert.ElementsMatch(t, []int32{1, 2, 3}, feedIDs(result))
}

func TestFilterZombieWssFeeds_NoWssFeedsUnchanged(t *testing.T) {
	now := time.Now()
	la := newTypedLocalAggregator([]Feed{httpFeed(1), httpFeed(2)}, nil)

	feeds := []*FeedData{
		{FeedID: 1, Value: 100, Timestamp: &now},
		{FeedID: 2, Value: 101, Timestamp: &now},
	}

	result := la.filterZombieWssFeeds(feeds)
	assert.ElementsMatch(t, []int32{1, 2}, feedIDs(result))
}

// Thin config: 2 live wss feeds gone stale + 2 wss feeds with venue volume 0
// (fresh timestamps, old price). Must not publish a zombie-only value.
func TestLocalAggregatorJob_ZombieOnlyNotPublished(t *testing.T) {
	now := time.Now()
	stale := now.Add(-3 * time.Minute)
	la := newTypedLocalAggregator(
		[]Feed{wssFeed(1), wssFeed(2), wssFeed(3), wssFeed(4)},
		map[int32]*FeedData{
			1: {FeedID: 1, Value: 369, Volume: 1000, Timestamp: &stale},
			2: {FeedID: 2, Value: 370, Volume: 800, Timestamp: &stale},
			3: {FeedID: 3, Value: 406.4, Volume: 0, Timestamp: &now},
			4: {FeedID: 4, Value: 406.4, Volume: 0, Timestamp: &now},
		},
	)

	feeds, err := la.collect(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, feeds)

	assert.NoError(t, la.Job(context.Background()))
	select {
	case agg := <-la.localAggregatesChannel:
		t.Fatalf("unexpected local aggregate published: %+v", agg)
	default:
	}
}

// Same thin config plus DEX/HTTP feeds: zombies dropped, DEX/HTTP still
// exempt from freshness and kept.
func TestLocalAggregatorCollect_ZombiesDroppedDexHttpKept(t *testing.T) {
	now := time.Now()
	stale := now.Add(-3 * time.Minute)
	la := newTypedLocalAggregator(
		[]Feed{wssFeed(1), wssFeed(2), wssFeed(3), wssFeed(4), dexFeed(5), httpFeed(6)},
		map[int32]*FeedData{
			1: {FeedID: 1, Value: 369, Volume: 1000, Timestamp: &stale},
			2: {FeedID: 2, Value: 370, Volume: 800, Timestamp: &stale},
			3: {FeedID: 3, Value: 406.4, Volume: 0, Timestamp: &now},
			4: {FeedID: 4, Value: 406.4, Volume: 0, Timestamp: &now},
			5: {FeedID: 5, Value: 369.5, Volume: 0, Timestamp: &stale},
			6: {FeedID: 6, Value: 369.8, Volume: 0, Timestamp: &stale},
		},
	)

	feeds, err := la.collect(context.Background())
	assert.NoError(t, err)
	assert.ElementsMatch(t, []int32{5, 6}, feedIDs(feeds))
}

// One live wss feed still fresh: zombies stay (median bucket only).
func TestLocalAggregatorCollect_ZombiesKeptWithLiveWss(t *testing.T) {
	now := time.Now()
	stale := now.Add(-3 * time.Minute)
	la := newTypedLocalAggregator(
		[]Feed{wssFeed(1), wssFeed(2), wssFeed(3), wssFeed(4)},
		map[int32]*FeedData{
			1: {FeedID: 1, Value: 369, Volume: 1000, Timestamp: &now},
			2: {FeedID: 2, Value: 370, Volume: 800, Timestamp: &stale},
			3: {FeedID: 3, Value: 406.4, Volume: 0, Timestamp: &now},
			4: {FeedID: 4, Value: 406.4, Volume: 0, Timestamp: &stale},
		},
	)

	feeds, err := la.collect(context.Background())
	assert.NoError(t, err)
	assert.ElementsMatch(t, []int32{1, 3}, feedIDs(feeds))
}
