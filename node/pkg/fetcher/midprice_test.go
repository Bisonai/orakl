package fetcher

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMidAggregator(priceSource string) *LocalAggregator {
	la := newLocalAggregatorWithFreshness(intPtr(60000))
	la.Config.PriceSource = priceSource
	return la
}

func bookFeed(id int32, last, bid, ask float64, tradeAge, bookAge time.Duration) *FeedData {
	now := time.Now()
	return &FeedData{
		FeedID: id, Value: last, Volume: 10,
		Timestamp: timePtr(now.Add(-tradeAge)),
		Bid:       bid, Ask: ask, BookTimestamp: timePtr(now.Add(-bookAge)),
	}
}

func TestSelectMidPrices_UsesMidOfValidBook(t *testing.T) {
	la := newMidAggregator(PriceSourceMid)
	src := bookFeed(1, 222, 220, 221, 10*time.Minute, 5*time.Second)

	out := la.selectMidPrices([]*FeedData{src})

	require.Len(t, out, 1)
	assert.InDelta(t, 220.5, out[0].Value, 1e-9)
	assert.True(t, out[0].FromBook)
	assert.Equal(t, src.BookTimestamp, out[0].Timestamp, "timestamp is the book time")
	assert.Equal(t, 10.0, out[0].Volume, "24h volume is kept for weighting")
	// shared source entry is not mutated
	assert.Equal(t, 222.0, src.Value)
	assert.False(t, src.FromBook)
}

func TestSelectMidPrices_FallsBackToLastTrade(t *testing.T) {
	la := newMidAggregator(PriceSourceMid)
	cases := map[string]*FeedData{
		"no book":         {FeedID: 1, Value: 100, Volume: 1, Timestamp: timePtr(time.Now())},
		"spread above 2%": bookFeed(2, 100, 98, 101, 0, 0),                 // 3.0%
		"stale book":      bookFeed(3, 100, 99.9, 100.1, 0, 6*time.Minute), // > MidBookMaxAge
		"crossed book":    bookFeed(4, 100, 101, 100, 0, 0),
		"one-sided book":  bookFeed(5, 100, 0, 100.1, 0, 0),
	}
	for name, fd := range cases {
		t.Run(name, func(t *testing.T) {
			out := la.selectMidPrices([]*FeedData{fd})
			require.Len(t, out, 1)
			assert.Same(t, fd, out[0], "feed passes through unchanged")
			assert.False(t, out[0].FromBook)
		})
	}
}

func TestSelectMidPrices_SpreadAtCapIsUsed(t *testing.T) {
	la := newMidAggregator(PriceSourceMid)
	// (ask-bid)/mid = 2/100 = 2% exactly
	out := la.selectMidPrices([]*FeedData{bookFeed(1, 99, 99, 101, 0, 0)})
	assert.True(t, out[0].FromBook)
	assert.InDelta(t, 100.0, out[0].Value, 1e-9)
}

func TestSelectMidPrices_DisabledByDefault(t *testing.T) {
	la := newMidAggregator("")
	fd := bookFeed(1, 222, 220, 221, 0, 0)
	out := la.selectMidPrices([]*FeedData{fd})
	assert.Same(t, fd, out[0], "last trade without priceSource mid")
}

func TestFilterStaleFeeds_BookMidUsesBookAge(t *testing.T) {
	la := newMidAggregator(PriceSourceMid)
	now := time.Now()
	feeds := []*FeedData{
		// book unchanged for 3 min: still a current price
		{FeedID: 1, Value: 220.5, Volume: 1, Timestamp: timePtr(now.Add(-3 * time.Minute)), FromBook: true},
		// last trade 3 min old: stale under feedDataFreshness 60 s
		{FeedID: 2, Value: 214, Volume: 1, Timestamp: timePtr(now.Add(-3 * time.Minute))},
		{FeedID: 3, Value: 215, Volume: 1, Timestamp: timePtr(now)},
	}
	out := la.filterStaleFeeds(feeds)
	ids := []int32{}
	for _, fd := range out {
		ids = append(ids, fd.FeedID)
	}
	assert.Equal(t, []int32{1, 3}, ids)
}

func TestCollect_MidThenStaleFilter(t *testing.T) {
	now := time.Now()
	data := &LatestFeedDataMap{FeedDataMap: map[int32]*FeedData{
		// quiet upbit: last trade 10 min old, live book -> mid
		1: bookFeed(1, 214, 212, 213, 10*time.Minute, 20*time.Second),
		// quiet CEX without book data: last trade 10 min old -> dropped
		2: {FeedID: 2, Value: 230, Volume: 5, Timestamp: timePtr(now.Add(-10 * time.Minute))},
		// active CEX without book data -> last trade kept
		3: {FeedID: 3, Value: 213, Volume: 5, Timestamp: timePtr(now)},
		// DEX (volume 0) -> unchanged
		4: {FeedID: 4, Value: 212.7, Timestamp: timePtr(now.Add(-10 * time.Minute))},
	}}
	la := newMidAggregator(PriceSourceMid)
	la.Feeds = []Feed{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}}
	la.latestFeedDataMap = data

	out, err := la.collect(context.Background())
	require.NoError(t, err)

	got := map[int32]float64{}
	for _, fd := range out {
		got[fd.FeedID] = fd.Value
	}
	assert.Equal(t, map[int32]float64{1: 212.5, 3: 213, 4: 212.7}, got)
}

func bookOnlyFeed(id int32, bid, ask float64, bookAge time.Duration) *FeedData {
	ts := timePtr(time.Now().Add(-bookAge))
	return &FeedData{
		FeedID: id, Value: (bid + ask) / 2, Volume: 10, Timestamp: ts,
		Bid: bid, Ask: ask, BookTimestamp: ts, BookOnly: true,
	}
}

func TestSelectMidPrices_BookOnly(t *testing.T) {
	la := newMidAggregator(PriceSourceMid)
	fd := bookOnlyFeed(1, 220, 221, 5*time.Second)
	out := la.selectMidPrices([]*FeedData{fd})
	require.Len(t, out, 1)
	assert.InDelta(t, 220.5, out[0].Value, 1e-9)
	assert.True(t, out[0].FromBook)
	assert.Equal(t, 10.0, out[0].Volume)

	// no last trade to fall back on: dropped when the mid is unusable
	for name, fd := range map[string]*FeedData{
		"spread above 2%": bookOnlyFeed(2, 98, 101, 0),
		"stale book":      bookOnlyFeed(3, 99.9, 100.1, 6*time.Minute),
	} {
		t.Run(name, func(t *testing.T) {
			assert.Empty(t, la.selectMidPrices([]*FeedData{fd}))
		})
	}
}

func TestSelectMidPrices_BookOnlyIgnoredWithoutPriceSourceMid(t *testing.T) {
	la := newMidAggregator("")
	trade := bookFeed(1, 222, 220, 221, 0, 0)
	out := la.selectMidPrices([]*FeedData{trade, bookOnlyFeed(2, 220, 221, 0)})
	require.Len(t, out, 1, "book-only feed is absent until its first trade")
	assert.Same(t, trade, out[0])
}
