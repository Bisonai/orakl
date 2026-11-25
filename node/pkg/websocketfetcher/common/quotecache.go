package common

import (
	"sync"
	"time"
)

// BookEmitInterval is the minimum time between two book-driven emits of the
// same feed. Tick-by-tick book streams (okx bbo-tbt, bybit orderbook.1, ...)
// push hundreds of updates per second on liquid pairs; the aggregator reads a
// feed's latest value every aggregate interval, so emitting every update only
// costs CPU. Every update is still recorded and carried by the next emit
// (book or ticker).
const BookEmitInterval = 250 * time.Millisecond

// QuoteCache merges a provider's two streams per feed: the ticker (last trade
// price, 24h volume, trade time) and the order book (best bid/ask, book
// time). LatestFeedDataMap replaces a feed's entry on every update, so each
// emitted FeedData must carry both halves.
type QuoteCache struct {
	mu           sync.Mutex
	feeds        map[int32]*FeedData
	lastBookEmit map[int32]time.Time
	now          func() time.Time
	// bookOnlyVolume, if set, returns a feed's known 24h volume without a
	// ticker (see NewQuoteCacheWithVolumes).
	bookOnlyVolume func(id int32) (float64, bool)
}

func NewQuoteCache() *QuoteCache {
	return &QuoteCache{feeds: make(map[int32]*FeedData), lastBookEmit: make(map[int32]time.Time), now: time.Now}
}

// NewQuoteCacheWithVolumes returns a QuoteCache that also emits book-only
// quotes for feeds without a ticker yet whose 24h volume is in volumes (a
// provider's REST-polled volume cache), so sparse-trade venues emit from the
// book without waiting for the first trade.
func NewQuoteCacheWithVolumes(volumes *VolumeCacheMap) *QuoteCache {
	c := NewQuoteCache()
	c.bookOnlyVolume = volumes.FreshVolume
	return c
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
// trade time come from the ticker) and were not book-emitted within
// BookEmitInterval.
//
// A feed without a ticker yet emits a book-only quote (BookOnly, Value = mid,
// Timestamp = book time) if the cache was built with
// NewQuoteCacheWithVolumes, its 24h volume is known and positive and the
// book is two-sided and not crossed. Otherwise it emits nothing: without a
// 24h volume it cannot be weighted, and a volume-0 feed would be treated as
// a DEX feed.
func (c *QuoteCache) ApplyBook(ids []int32, bid, ask float64, timestamp time.Time) []*FeedData {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	result := make([]*FeedData, 0, len(ids))
	for _, id := range ids {
		prev, ok := c.feeds[id]
		if !ok {
			prev = &FeedData{FeedID: id}
			c.feeds[id] = prev
		}
		ts := timestamp
		prev.Bid, prev.Ask, prev.BookTimestamp = bid, ask, &ts
		out := *prev
		if prev.Timestamp == nil {
			volume, ok := c.bookOnlyQuoteVolume(id, bid, ask)
			if !ok {
				continue
			}
			out.Value, out.Volume, out.Timestamp, out.BookOnly = (bid+ask)/2, volume, &ts, true
		}
		if last, ok := c.lastBookEmit[id]; ok && now.Sub(last) < BookEmitInterval {
			continue
		}
		c.lastBookEmit[id] = now
		result = append(result, &out)
	}
	return result
}

func (c *QuoteCache) bookOnlyQuoteVolume(id int32, bid, ask float64) (float64, bool) {
	if c.bookOnlyVolume == nil || bid <= 0 || ask < bid {
		return 0, false
	}
	volume, ok := c.bookOnlyVolume(id)
	return volume, ok && volume > 0
}

// SetTickerBook sets a ticker's own best bid/ask on its FeedData, for
// providers whose ticker stream carries the book and also pushes on book
// changes (no separate book stream to merge). A missing or one-sided book is
// left unset.
func SetTickerBook(data *FeedData, bid, ask float64, timestamp time.Time) {
	if bid <= 0 || ask <= 0 {
		return
	}
	ts := timestamp
	data.Bid, data.Ask, data.BookTimestamp = bid, ask, &ts
}
