package types

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Proxy struct {
	ID       int64   `db:"id"`
	Protocol string  `db:"protocol"`
	Host     string  `db:"host"`
	Port     int     `db:"port"`
	Location *string `db:"location"`
}

func (proxy *Proxy) GetProxyUrl() string {
	return fmt.Sprintf("%s://%s:%d", proxy.Protocol, proxy.Host, proxy.Port)
}

type Feed struct {
	ID         int32           `db:"id"`
	Name       string          `db:"name"`
	Definition json.RawMessage `db:"definition"`
	ConfigID   int32           `db:"config_id"`
}

type FeedData struct {
	FeedID    int32      `db:"feed_id"`
	Value     float64    `db:"value"`
	Volume    float64    `db:"volume"`
	Timestamp *time.Time `db:"timestamp"`

	// Best bid/ask of the source's order book, for providers that stream it
	// (zero when the provider has no book data). BookTimestamp is the time
	// of the last book update. With priceSource "mid" the local aggregator
	// uses the mid of a valid book instead of Value (the last trade).
	Bid           float64    `db:"-"`
	Ask           float64    `db:"-"`
	BookTimestamp *time.Time `db:"-"`

	// FromBook is set on the aggregator's copy when Value is the book mid;
	// the stale filter then applies the book age limit instead of the
	// config's feedDataFreshness.
	FromBook bool `db:"-"`

	// BookOnly marks a book-driven emit of a feed that has no trade yet
	// (sparse-trade venues, after a (re)start): Value is the book mid,
	// Timestamp the book time and Volume the provider's REST 24h volume.
	// Only priceSource "mid" aggregators use it, and only while its mid is
	// valid; others treat the feed as absent until its first trade.
	BookOnly bool `db:"-"`
}

type LocalAggregate struct {
	ConfigID  int32     `db:"config_id" json:"configId"`
	Value     int64     `db:"value" json:"value"`
	Timestamp time.Time `db:"timestamp" json:"timestamp"`
}

type GlobalAggregate struct {
	ConfigID  int32     `db:"config_id" json:"configId"`
	Value     int64     `db:"value" json:"value"`
	Round     int32     `db:"round" json:"round"`
	Timestamp time.Time `db:"timestamp" json:"timestamp"`
}

type Proof struct {
	ConfigID int32  `db:"config_id" json:"configId"`
	Round    int32  `db:"round" json:"round"`
	Proof    []byte `db:"proof" json:"proof"`
}

type Config struct {
	ID                  int32  `db:"id" json:"id"`
	Name                string `db:"name" json:"name"`
	FetchInterval       *int   `db:"fetch_interval" json:"fetchInterval"`
	AggregateInterval   *int   `db:"aggregate_interval" json:"aggregateInterval"`
	SubmitInterval      *int   `db:"submit_interval" json:"submitInterval"`
	Decimals            *int   `db:"decimals" json:"decimals"`
	FeedDataFreshness *int `db:"feed_data_freshness" json:"feedDataFreshness"`
}

type LatestFeedDataMap struct {
	FeedDataMap map[int32]*FeedData
	Mu          sync.RWMutex
}

func (m *LatestFeedDataMap) GetLatestFeedData(feedIds []int32) ([]*FeedData, error) {
	result := make([]*FeedData, 0, len(feedIds))
	m.Mu.RLock()
	defer m.Mu.RUnlock()
	for _, feedId := range feedIds {
		feedData, ok := m.FeedDataMap[feedId]
		if ok {
			result = append(result, feedData)
		}
	}
	return result, nil
}

func (m *LatestFeedDataMap) SetLatestFeedData(feedData []*FeedData) error {
	if len(feedData) == 0 {
		return nil
	}

	m.Mu.Lock()
	defer m.Mu.Unlock()
	for _, data := range feedData {
		if data == nil {
			continue
		}

		prev, ok := m.FeedDataMap[data.FeedID]
		switch {
		case ok && data.BookOnly && !prev.BookOnly:
			// The feed already has a trade (e.g. from before a fetcher
			// refresh): keep it and take only the newer book.
			if data.BookTimestamp != nil && (prev.BookTimestamp == nil || data.BookTimestamp.After(*prev.BookTimestamp)) {
				merged := *prev
				merged.Bid, merged.Ask, merged.BookTimestamp = data.Bid, data.Ask, data.BookTimestamp
				m.FeedDataMap[data.FeedID] = &merged
			}
			continue
		case ok && prev.BookOnly && !data.BookOnly:
			// A trade supersedes a book-only entry even if the trade time
			// is before the last book time.
		case ok && prev.Timestamp.After(*data.Timestamp):
			continue
		}
		m.FeedDataMap[data.FeedID] = data
	}
	return nil
}

func (m *LatestFeedDataMap) CleanupJob(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.Mu.Lock()
			for k, v := range m.FeedDataMap {
				if v.Timestamp != nil && v.Timestamp.Before(time.Now().Add(-6*time.Hour)) {
					delete(m.FeedDataMap, k)
				}
			}
			m.Mu.Unlock()
		}
	}
}
