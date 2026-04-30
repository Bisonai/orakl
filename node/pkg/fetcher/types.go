package fetcher

import (
	"context"
	"math/big"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/bus"
	"bisonai.com/miko/node/pkg/common/types"
	"bisonai.com/miko/node/pkg/utils/reducer"
	"bisonai.com/miko/node/pkg/websocketfetcher"
)

const (
	SelectAllProxiesQuery                 = `SELECT * FROM proxies`
	SelectConfigsQuery                    = `SELECT id, name, fetch_interval, decimals, feed_data_freshness, multiply_by, multiply_by_reciprocal FROM configs`
	SelectHttpRequestFeedsByConfigIdQuery = `SELECT * FROM feeds WHERE config_id = @config_id AND NOT (definition::jsonb ? 'type')`
	SelectFeedsByConfigIdQuery            = `SELECT * FROM feeds WHERE config_id = @config_id`
	InsertLocalAggregateQuery             = `INSERT INTO local_aggregates (config_id, value) VALUES (@config_id, @value)`
	DECIMALS                              = 8
	DefaultFeedDataDumpInterval           = time.Second * 10
	ForeignExchangePricePairs             = "GBP-USD,EUR-USD,KRW-USD,JPY-USD,CHF-USD"
	DefaultMedianRatio                    = 0.05
	LocalAggregatesChannelSize            = 2_000
	DefaultLocalAggregateInterval         = 200 * time.Millisecond
	DefaultFeedDataDumpChannelSize        = 20000
	MaxOutlierRemovalRatio                = 0.25
)

type Feed = types.Feed
type FeedData = types.FeedData
type LocalAggregate = types.LocalAggregate
type Proxy = types.Proxy
type LatestFeedDataMap = types.LatestFeedDataMap

type Config struct {
	ID                    int32   `db:"id"`
	Name                  string  `db:"name"`
	FetchInterval         int32   `db:"fetch_interval"`
	Decimals              *int    `db:"decimals"`
	FeedDataFreshness     *int    `db:"feed_data_freshness"`
	MultiplyBy            *string `db:"multiply_by"`
	MultiplyByReciprocal  bool    `db:"multiply_by_reciprocal"`
}

// LocalAggregateValueMap is a process-wide cache of the most recent raw
// aggregate value (pre-decimals) for each config, keyed by config name.
//
// It exists to support synthetic configs whose value is derived by
// multiplying their own aggregated feed value by another config's
// aggregate (e.g. STG-KRW = STG-USDT * KRW-USD).  The value is the
// floating-point aggregate emitted by LocalAggregator, before
// applyDecimals is applied, so consumers can multiply directly without
// having to reverse the decimal scaling.
type LocalAggregateValueMap struct {
	Mu   sync.RWMutex
	Data map[string]LocalAggregateValueEntry
}

type LocalAggregateValueEntry struct {
	Value     float64
	Timestamp time.Time
}

func NewLocalAggregateValueMap() *LocalAggregateValueMap {
	return &LocalAggregateValueMap{
		Data: make(map[string]LocalAggregateValueEntry),
	}
}

func (m *LocalAggregateValueMap) Set(name string, value float64) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.Data[name] = LocalAggregateValueEntry{Value: value, Timestamp: time.Now()}
}

func (m *LocalAggregateValueMap) Get(name string) (LocalAggregateValueEntry, bool) {
	m.Mu.RLock()
	defer m.Mu.RUnlock()
	v, ok := m.Data[name]
	return v, ok
}

type Fetcher struct {
	Config
	Feeds []Feed

	fetcherCtx          context.Context
	cancel              context.CancelFunc
	isRunning           bool
	latestFeedDataMap   *LatestFeedDataMap
	FeedDataDumpChannel chan *FeedData
	circuitBreakers     *circuitBreakerMap
}

type LocalAggregator struct {
	Config
	Feeds []Feed

	aggregatorCtx context.Context
	cancel        context.CancelFunc
	isRunning     bool
	bus           *bus.MessageBus

	localAggregatesChannel chan *LocalAggregate
	latestFeedDataMap      *LatestFeedDataMap
	localAggregateValueMap *LocalAggregateValueMap
}

type FeedDataBulkWriter struct {
	Interval time.Duration

	FeedDataDumpChannel chan *FeedData
	writerCtx           context.Context
	cancel              context.CancelFunc
	isRunning           bool
}

type LocalAggregateBulkWriter struct {
	Interval time.Duration

	bulkWriterCtx          context.Context
	cancel                 context.CancelFunc
	isRunning              bool
	localAggregatesChannel chan *LocalAggregate
}

type App struct {
	Bus                      *bus.MessageBus
	Fetchers                 map[int32]*Fetcher
	LocalAggregators         map[int32]*LocalAggregator
	FeedDataBulkWriter       *FeedDataBulkWriter
	LocalAggregateBulkWriter *LocalAggregateBulkWriter
	WebsocketFetcher         *websocketfetcher.App
	LatestFeedDataMap        *LatestFeedDataMap
	LocalAggregateValueMap   *LocalAggregateValueMap
	Proxies                  []Proxy
	FeedDataDumpChannel      chan *FeedData
}

type Definition struct {
	Url      *string           `json:"url"`
	Headers  map[string]string `json:"headers"`
	Method   *string           `json:"method"`
	Reducers []reducer.Reducer `json:"reducers"`
	Location *string           `json:"location"`

	// dex specific
	Type           *string `json:"type"`
	ChainID        *string `json:"chainId"`
	Address        *string `json:"address"`
	PoolID         *string `json:"poolId"`
	Token0Decimals *int64  `json:"token0Decimals"`
	Token1Decimals *int64  `json:"token1Decimals"`
	Reciprocal     *bool   `json:"reciprocal"`

	// Per-feed synthetic multiplier. When MultiplyBy names another config,
	// the LocalAggregator multiplies (or divides, when MultiplyByReciprocal
	// is true) this feed's value by that config's most recent raw aggregate
	// before it joins this config's aggregation. Used when a single source
	// reports the wrong quote currency — e.g. PancakeSwap on Base reports
	// IDRX/USDC but the aggregating config is IDRX-USDT, so the feed
	// multiplies its value by USDC-USDT to land in the right pair.
	MultiplyBy           *string `json:"multiplyBy"`
	MultiplyByReciprocal *bool   `json:"multiplyByReciprocal"`
}

type ChainHelper interface {
	ReadContract(ctx context.Context, contractAddress string, functionString string, args ...interface{}) (interface{}, error)
	ChainID() *big.Int
	Close()
}
