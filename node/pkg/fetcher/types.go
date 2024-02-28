package fetcher

import (
	"context"
	"encoding/json"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/utils"
)

const (
	SelectActiveAdaptersQuery   = `SELECT * FROM adapters WHERE active = true`
	SelectFeedsByAdapterIdQuery = `SELECT * FROM feeds WHERE adapter_id = @adapterId`
	InsertLocalAggregateQuery   = `INSERT INTO local_aggregates (name, value) VALUES (@name, @value)`
)

type Adapter struct {
	ID     int64  `db:"id"`
	Name   string `db:"name"`
	Active bool   `db:"active"`
}

type Fetcher struct {
	Adapter
	Feeds []Feed

	adapterCtx context.Context
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
	Bus      *bus.MessageBus
	Fetchers map[int64]*Fetcher
}

type Definition struct {
	Url      string            `json:"url"`
	Headers  map[string]string `json:"headers"`
	Method   string            `json:"method"`
	Reducers []utils.Reducer   `json:"reducers"`
}

type Aggregate struct {
	Name      string     `db:"name"`
	Value     int64      `db:"value"`
	Timestamp *time.Time `db:"timestamp"`
}

type redisAggregate struct {
	Value     int64     `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}
