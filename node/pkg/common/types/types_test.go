package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMap() *LatestFeedDataMap {
	return &LatestFeedDataMap{FeedDataMap: make(map[int32]*FeedData)}
}

func at(t time.Time) *time.Time { return &t }

func TestSetLatestFeedData_TradeReplacesBookOnlyEvenIfOlder(t *testing.T) {
	m := newMap()
	now := time.Now()
	bookOnly := &FeedData{FeedID: 1, Value: 100, Volume: 5, Timestamp: at(now), Bid: 99, Ask: 101, BookTimestamp: at(now), BookOnly: true}
	require.NoError(t, m.SetLatestFeedData([]*FeedData{bookOnly}))
	assert.Same(t, bookOnly, m.FeedDataMap[1])

	trade := &FeedData{FeedID: 1, Value: 100.5, Volume: 5, Timestamp: at(now.Add(-time.Millisecond)), Bid: 99, Ask: 101, BookTimestamp: at(now)}
	require.NoError(t, m.SetLatestFeedData([]*FeedData{trade}))
	assert.Same(t, trade, m.FeedDataMap[1], "trade time before the book time")

	// later book emits of the cache carry the trade time
	bookEmit := *trade
	bookEmit.BookTimestamp = at(now.Add(time.Second))
	require.NoError(t, m.SetLatestFeedData([]*FeedData{&bookEmit}))
	assert.Same(t, &bookEmit, m.FeedDataMap[1])
}

func TestSetLatestFeedData_BookOnlyKeepsExistingTrade(t *testing.T) {
	m := newMap()
	now := time.Now()
	trade := &FeedData{FeedID: 1, Value: 100.5, Volume: 5, Timestamp: at(now.Add(-time.Minute)), Bid: 99, Ask: 101, BookTimestamp: at(now.Add(-time.Minute))}
	require.NoError(t, m.SetLatestFeedData([]*FeedData{trade}))

	// e.g. a fresh QuoteCache after a fetcher refresh
	bookOnly := &FeedData{FeedID: 1, Value: 100, Volume: 6, Timestamp: at(now), Bid: 99.5, Ask: 100.5, BookTimestamp: at(now), BookOnly: true}
	require.NoError(t, m.SetLatestFeedData([]*FeedData{bookOnly}))
	got := m.FeedDataMap[1]
	assert.False(t, got.BookOnly)
	assert.Equal(t, 100.5, got.Value, "trade price kept")
	assert.Equal(t, 5.0, got.Volume)
	assert.True(t, got.Timestamp.Equal(*trade.Timestamp), "trade time kept")
	assert.Equal(t, 99.5, got.Bid)
	assert.Equal(t, 100.5, got.Ask)
	assert.True(t, got.BookTimestamp.Equal(now))
	assert.Equal(t, 99.0, trade.Bid, "stored entry is not mutated")

	// an older book is ignored
	older := *bookOnly
	older.Bid, older.BookTimestamp = 1, at(now.Add(-time.Second))
	require.NoError(t, m.SetLatestFeedData([]*FeedData{&older}))
	assert.Equal(t, 99.5, m.FeedDataMap[1].Bid)
}

func TestSetLatestFeedData_BookOnlyReplacesOlderBookOnly(t *testing.T) {
	m := newMap()
	now := time.Now()
	first := &FeedData{FeedID: 1, Value: 100, Volume: 5, Timestamp: at(now), BookTimestamp: at(now), BookOnly: true}
	second := &FeedData{FeedID: 1, Value: 101, Volume: 5, Timestamp: at(now.Add(time.Second)), BookTimestamp: at(now.Add(time.Second)), BookOnly: true}
	require.NoError(t, m.SetLatestFeedData([]*FeedData{second, first}))
	assert.Same(t, second, m.FeedDataMap[1], "older book-only is dropped")
}
