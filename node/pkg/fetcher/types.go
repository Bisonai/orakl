package fetcher

import (
	"context"
	"math/big"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/common/types"
	"bisonai.com/orakl/node/pkg/utils/reducer"
	"bisonai.com/orakl/node/pkg/websocketfetcher"
)

const (
	SelectAllProxiesQuery                 = `SELECT * FROM proxies`
	SelectConfigsQuery                    = `SELECT id, name, fetch_interval FROM configs`
	SelectHttpRequestFeedsByConfigIdQuery = `SELECT * FROM feeds WHERE config_id = @config_id AND NOT (definition::jsonb ? 'type')`
	SelectFeedsByConfigIdQuery            = `SELECT * FROM feeds WHERE config_id = @config_id`
	InsertLocalAggregateQuery             = `INSERT INTO local_aggregates (config_id, value) VALUES (@config_id, @value)`
	DECIMALS                              = 8
	DefaultStreamInterval                 = time.Second * 5
	ForeignExchangePricePairs             = "GBP-USD,EUR-USD,KRW-USD,JPY-USD,CHF-USD"
	DefaultMedianRatio                    = 0.05
)

type Feed = types.Feed
type FeedData = types.FeedData
type LocalAggregate = types.LocalAggregate
type Proxy = types.Proxy
type LatestFeedDataMap = types.LatestFeedDataMap

type Config struct {
	ID            int32  `db:"id"`
	Name          string `db:"name"`
	FetchInterval int32  `db:"fetch_interval"`
}

type Fetcher struct {
	Config
	Feeds []Feed

	fetcherCtx        context.Context
	cancel            context.CancelFunc
	isRunning         bool
	latestFeedDataMap *LatestFeedDataMap
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
}

type FeedDataBulkWriter struct {
	Interval time.Duration

	writerCtx context.Context
	cancel    context.CancelFunc
	isRunning bool
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
	Proxies                  []Proxy
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
	Token0Decimals *int64  `json:"token0Decimals"`
	Token1Decimals *int64  `json:"token1Decimals"`
	Reciprocal     *bool   `json:"reciprocal"`
}

type ChainHelper interface {
	ReadContract(ctx context.Context, contractAddress string, functionString string, args ...interface{}) (interface{}, error)
	ChainID() *big.Int
	Close()
}
