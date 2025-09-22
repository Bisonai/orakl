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
	ID                int32  `db:"id" json:"id"`
	Name              string `db:"name" json:"name"`
	FetchInterval     int    `db:"fetch_interval" json:"fetchInterval"`
	AggregateInterval int    `db:"aggregate_interval" json:"aggregateInterval"`
	SubmitInterval    int    `db:"submit_interval" json:"submitInterval"`
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
		if ok && prev.Timestamp.After(*data.Timestamp) {
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
				if v.Timestamp != nil && v.Timestamp.Before(time.Now().Add(-24*time.Hour)) {
					delete(m.FeedDataMap, k)
				}
			}
			m.Mu.Unlock()
		}
	}
}
