package aggregator

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/raft"
)

const (
	RoundSync                       raft.MessageType = "roundSync"
	PriceData                       raft.MessageType = "priceData"
	SelectActiveAggregatorsQuery                     = `SELECT * FROM aggregators WHERE active = true`
	SelectLatestLocalAggregateQuery                  = `SELECT * FROM local_aggregates WHERE name = @name ORDER BY timestamp DESC LIMIT 1`
	InsertGlobalAggregateQuery                       = `INSERT INTO global_aggregates (name, value, round) VALUES (@name, @value, @round)`
)

type redisAggregate struct {
	Value     int64     `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

type App struct {
	Bus         *bus.MessageBus
	Aggregators map[int64]*AggregatorNode
}

type Aggregator struct {
	ID     int64  `db:"id"`
	Name   string `db:"name"`
	Active bool   `db:"active"`
}

type AggregatorNode struct {
	Aggregator
	Raft *raft.Raft

	LeaderJobTicker *time.Ticker
	JobTicker       *time.Ticker

	LeaderJobTimeout *time.Duration
	JobTimeout       *time.Duration

	CollectedPrices map[int][]int
	AggregatorMutex sync.Mutex

	RoundID int

	nodeCtx    context.Context
	nodeCancel context.CancelFunc
	isRunning  bool
}

type RoundSyncMessage struct {
	LeaderID string `json:"leaderID"`
	RoundID  int    `json:"roundID"`
}

type PriceDataMessage struct {
	RoundID   int `json:"roundID"`
	PriceData int `json:"priceData"`
}

func GetLatestAggregateFromRdb(ctx context.Context, name string) (redisAggregate, error) {
	key := "latestAggregate:" + name
	var aggregate redisAggregate
	data, err := db.Get(ctx, key)
	if err != nil {
		return aggregate, err
	}

	err = json.Unmarshal([]byte(data), &aggregate)
	if err != nil {
		return aggregate, err
	}
	return aggregate, nil
}
