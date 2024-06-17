package common

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/common/types"
	"bisonai.com/orakl/node/pkg/wss"
)

const (
	DECIMALS                  = 8
	GetAllWebsocketFeedsQuery = `SELECT *
	FROM feeds
	WHERE definition @> '{"type": "wss"}';`
	GetAllProxiesQuery  = `SELECT * FROM proxies`
	VolumeCacheLifespan = 10 * time.Minute
	VolumeFetchInterval = 10000
)

type Proxy types.Proxy

func (proxy *Proxy) GetProxyUrl() string {
	return fmt.Sprintf("%s://%s:%d", proxy.Protocol, proxy.Host, proxy.Port)
}

type Feed struct {
	ID         int32           `db:"id"`
	Name       string          `db:"name"`
	Definition json.RawMessage `db:"definition"`
	ConfigID   int32           `db:"config_id"`
}

type FeedData types.FeedData

type FeedDefinition struct {
	Type     string `json:"type"`
	Provider string `json:"provider"`
	Base     string `json:"base"`
	Quote    string `json:"quote"`
}

type FetcherConfig struct {
	FeedMaps       FeedMaps
	Proxy          string
	FeedDataBuffer chan FeedData
}

type FeedMaps struct {
	Combined  map[string]int32
	Separated map[string]int32
}

type FetcherOption func(*FetcherConfig)

func WithFeedMaps(feedMaps FeedMaps) FetcherOption {
	return func(c *FetcherConfig) {
		c.FeedMaps = feedMaps
	}
}

func WithProxy(proxy string) FetcherOption {
	return func(c *FetcherConfig) {
		c.Proxy = proxy
	}
}

func WithFeedDataBuffer(feedDataBuffer chan FeedData) FetcherOption {
	return func(c *FetcherConfig) {
		c.FeedDataBuffer = feedDataBuffer
	}
}

type Fetcher struct {
	FeedMap        map[string]int32
	Ws             *wss.WebsocketHelper
	FeedDataBuffer chan FeedData
	VolumeCacheMap VolumeCacheMap
}

type FetcherInterface interface {
	Run(context.Context)
}

type VolumeCache struct {
	UpdatedAt time.Time
	Volume    float64
}

type VolumeCacheMap struct {
	Map   map[int32]VolumeCache
	Mutex sync.Mutex
}
