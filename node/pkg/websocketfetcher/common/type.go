package common

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bisonai.com/orakl/node/pkg/wss"
)

const (
	DECIMALS           = 8
	GetAllFeedsQuery   = `SELECT * FROM feeds`
	GetAllProxiesQuery = `SELECT * FROM proxies`
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
	FeedId    int32      `db:"feed_id"`
	Value     float64    `db:"value"`
	Timestamp *time.Time `db:"timestamp"`
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
}

type FetcherInterface interface {
	Run(context.Context)
}
