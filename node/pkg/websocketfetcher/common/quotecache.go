package common

import (
	"sync"
	"time"
)

// QuoteCache merges a provider's two streams per feed: the ticker (last trade
// price, 24h volume, trade time) and the order book (best bid/ask, book
// time). LatestFeedDataMap replaces a feed's entry on every update, so each
// emitted FeedData must carry both halves.
type QuoteCache struct {
	mu    sync.Mutex
	feeds map[int32]*FeedData
}

func NewQuoteCache() *QuoteCache {
	return &QuoteCache{feeds: make(map[int32]*FeedData)}
}

// ApplyTrade records a ticker update and returns it with the feed's latest
// book attached.
func (c *QuoteCache) ApplyTrade(data *FeedData) *FeedData {
	c.mu.Lock()
	defer c.mu.Unlock()
	merged := *data
	if prev, ok := c.feeds[data.FeedID]; ok {
		merged.Bid, merged.Ask, merged.BookTimestamp = prev.Bid, prev.Ask, prev.BookTimestamp
	}
	c.feeds[data.FeedID] = &merged
	out := merged
	return &out
}

// ApplyBook records a top-of-book update for the feed IDs and returns the
// merged FeedData of feeds that already have a ticker (price, volume and
// trade time come from the ticker). Feeds without a ticker yet emit nothing:
// without a 24h volume they cannot be weighted.
func (c *QuoteCache) ApplyBook(ids []int32, bid, ask float64, timestamp time.Time) []*FeedData {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]*FeedData, 0, len(ids))
	for _, id := range ids {
		prev, ok := c.feeds[id]
		if !ok {
			prev = &FeedData{FeedID: id}
			c.feeds[id] = prev
		}
		ts := timestamp
		prev.Bid, prev.Ask, prev.BookTimestamp = bid, ask, &ts
		if prev.Timestamp == nil {
			continue
		}
		out := *prev
		result = append(result, &out)
	}
	return result
}
