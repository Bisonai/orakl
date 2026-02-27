//nolint:all
package fetcher

import (
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

func TestFilterStaleFeeds_MixFreshAndStale(t *testing.T) {
	la := newLocalAggregatorWithFreshness(intPtr(60000)) // 60 seconds

	now := time.Now()
	fresh1 := now.Add(-10 * time.Second)
	fresh2 := now.Add(-30 * time.Second)
	stale1 := now.Add(-2 * time.Minute)
	stale2 := now.Add(-5 * time.Minute)

	feeds := []*FeedData{
		{FeedID: 1, Value: 100.0, Timestamp: &fresh1},
		{FeedID: 2, Value: 0.0119, Timestamp: &stale1}, // stuck old price
		{FeedID: 3, Value: 101.0, Timestamp: &fresh2},
		{FeedID: 4, Value: 0.0119, Timestamp: &stale2}, // stuck old price
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
		{FeedID: 1, Value: 100.0, Timestamp: &stale1},
		{FeedID: 2, Value: 101.0, Timestamp: &stale2},
		{FeedID: 3, Value: 102.0, Timestamp: &stale3},
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
		{FeedID: 1, Value: 100.0, Timestamp: &fresh},
		{FeedID: 2, Value: 101.0, Timestamp: nil},   // nil timestamp should pass
		{FeedID: 3, Value: 102.0, Timestamp: &stale}, // stale
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
		{FeedID: 1, Value: 100.0, Timestamp: &justUnder},
		{FeedID: 2, Value: 101.0, Timestamp: &justOver},
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
