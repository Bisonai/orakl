package common

import (
	"context"
	"fmt"
	"sync"
	"time"
	"unicode"

	"bisonai.com/miko/node/pkg/chain/websocketchainreader"
	"bisonai.com/miko/node/pkg/common/types"
	"bisonai.com/miko/node/pkg/wss"
)

const (
	DECIMALS                  = 8
	GetAllWebsocketFeedsQuery = `SELECT *
	FROM public.feeds
	WHERE definition @> '{"type": "wss"}';`
	GetAllProxiesQuery  = `SELECT * FROM proxies`
	VolumeCacheLifespan = 10 * time.Minute
	VolumeFetchInterval = 10000
	VolumeFetchTimeout  = 6 * time.Second
)

type Feed = types.Feed
type FeedData = types.FeedData

func GetDexFeedsQuery(name string) string {
	name = capitalizeFirstLetter(name)
	return fmt.Sprintf(`SELECT * FROM public.feeds WHERE definition::jsonb @> '{"type": "%sPool"}'::jsonb;`, name)
}

type FeedDefinition struct {
	Type     string `json:"type"`
	Provider string `json:"provider"`
	Base     string `json:"base"`
	Quote    string `json:"quote"`
}

type DexFeedDefinition struct {
	Type           string `json:"type"`
	Address        string `json:"address"`
	ChainId        string `json:"chainId"`
	Token0Decimals int    `json:"token0Decimals"`
	Token1Decimals int    `json:"token1Decimals"`
	Reciprocal     *bool  `json:"reciprocal"`
}

type DexFeedDefinitionCapybara struct {
	DexFeedDefinition
	Token0Address string `json:"token0Address"`
	Token1Address string `json:"token1Address"`
	InitAmount    int64  `json:"initAmount"`
}

type FetcherConfig struct {
	FeedMaps       FeedMaps
	Proxy          string
	FeedDataBuffer chan *FeedData
}

type DexFetcherConfig struct {
	Feeds                []Feed
	WebsocketChainReader *websocketchainreader.ChainReader
	FeedDataBuffer       chan *FeedData
}

type FeedMaps struct {
	Combined  map[string][]int32
	Separated map[string][]int32
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

func WithFeedDataBuffer(feedDataBuffer chan *FeedData) FetcherOption {
	return func(c *FetcherConfig) {
		c.FeedDataBuffer = feedDataBuffer
	}
}

type DexFetcherOption func(*DexFetcherConfig)

func WithFeeds(feeds []Feed) DexFetcherOption {
	return func(c *DexFetcherConfig) {
		c.Feeds = feeds
	}
}

func WithWebsocketChainReader(websocketChainReader *websocketchainreader.ChainReader) DexFetcherOption {
	return func(c *DexFetcherConfig) {
		c.WebsocketChainReader = websocketChainReader
	}
}

func WithDexFeedDataBuffer(feedDataBuffer chan *FeedData) DexFetcherOption {
	return func(c *DexFetcherConfig) {
		c.FeedDataBuffer = feedDataBuffer
	}
}

type Fetcher struct {
	FeedMap        map[string][]int32
	Ws             *wss.WebsocketHelper
	FeedDataBuffer chan *FeedData
	VolumeCacheMap VolumeCacheMap
}

type DexFetcher struct {
	Feeds                []Feed
	WebsocketChainReader *websocketchainreader.ChainReader
	FeedDataBuffer       chan *FeedData
	LatestEntries        map[int32]*FeedData
	sync.Mutex
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

func (vc *VolumeCacheMap) Get(key int32) (VolumeCache, bool) {
	vc.Mutex.Lock()
	defer vc.Mutex.Unlock()

	cache, ok := vc.Map[key]
	return cache, ok
}

func (vc *VolumeCacheMap) Set(key int32, value VolumeCache) {
	vc.Mutex.Lock()
	defer vc.Mutex.Unlock()
	if vc.Map == nil {
		vc.Map = make(map[int32]VolumeCache)
	}

	vc.Map[key] = value
}

func capitalizeFirstLetter(s string) string {
	if len(s) == 0 {
		return s
	}

	// Convert string to a slice of runes
	runes := []rune(s)

	// Capitalize the first rune
	runes[0] = unicode.ToUpper(runes[0])

	// Convert back to string
	return string(runes)
}
