package fetcher

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/utils/reducer"
)

const (
	SelectAllProxiesQuery       = `SELECT * FROM proxies`
	SelectActiveAdaptersQuery   = `SELECT * FROM adapters WHERE active = true`
	SelectFeedsByAdapterIdQuery = `SELECT * FROM feeds WHERE adapter_id = @adapterId`
	InsertLocalAggregateQuery   = `INSERT INTO local_aggregates (name, value) VALUES (@name, @value)`
	DECIMALS                    = 8
)

type FeedData struct {
	FeedName string  `db:"name"`
	Value    float64 `db:"value"`
}

type Adapter struct {
	ID       int64  `db:"id"`
	Name     string `db:"name"`
	Active   bool   `db:"active"`
	Interval int    `db:"interval"`
}

type Proxy struct {
	ID       int64   `db:"id"`
	Protocol string  `db:"protocol"`
	Host     string  `db:"host"`
	Port     int     `db:"port"`
	Location *string `db:"location"`
}

type Fetcher struct {
	Adapter
	Feeds []Feed

	fetcherCtx context.Context
	cancel     context.CancelFunc
	isRunning  bool
}

type Feed struct {
	ID         int64           `db:"id"`
	Name       string          `db:"name"`
	Definition json.RawMessage `db:"definition"`
	AdapterID  int64           `db:"adapter_id"`
}

type App struct {
	Bus          *bus.MessageBus
	Fetchers     map[int64]*Fetcher
	Proxies      []Proxy
	ChainHelpers map[string]ChainHelper
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

type Aggregate struct {
	Name      string     `db:"name"`
	Value     int64      `db:"value"`
	Timestamp *time.Time `db:"timestamp"`
}

type FeedDataFromDB struct {
	AdapterID int64      `db:"adapter_id"`
	Name      string     `db:"name"`
	Value     float64    `db:"value"`
	Timestamp *time.Time `db:"timestamp"`
}

type redisAggregate struct {
	Value     int64     `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

type ChainHelper interface {
	ReadContract(ctx context.Context, contractAddress string, functionString string, args ...interface{}) (interface{}, error)
	ChainID() *big.Int
	Close()
}
