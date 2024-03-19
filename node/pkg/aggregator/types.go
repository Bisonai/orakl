package aggregator

import (
	"context"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/raft"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

const (
	RoundSync                        raft.MessageType = "roundSync"
	PriceData                        raft.MessageType = "priceData"
	SelectActiveAggregatorsQuery                      = `SELECT * FROM aggregators WHERE active = true`
	SelectLatestLocalAggregateQuery                   = `SELECT * FROM local_aggregates WHERE name = @name ORDER BY timestamp DESC LIMIT 1`
	InsertGlobalAggregateQuery                        = `INSERT INTO global_aggregates (name, value, round) VALUES (@name, @value, @round) RETURNING *`
	SelectLatestGlobalAggregateQuery                  = `SELECT * FROM global_aggregates WHERE name = @name ORDER BY round DESC LIMIT 1`
)

type redisLocalAggregate struct {
	Value     int64     `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

type redisGlobalAggregate struct {
	Value int64 `json:"value"`
	Round int64 `json:"round"`
}

type pgsLocalAggregate struct {
	Name      string    `db:"name"`
	Value     int64     `db:"value"`
	Timestamp time.Time `db:"timestamp"`
}

type globalAggregate struct {
	Name      string    `db:"name"`
	Value     int64     `db:"value"`
	Round     int64     `db:"round"`
	Timestamp time.Time `db:"timestamp"`
}

type App struct {
	Bus         *bus.MessageBus
	Aggregators map[int64]*AggregatorNode
	Host        host.Host
	Pubsub      *pubsub.PubSub
}

type Aggregator struct {
	ID     int64  `db:"id"`
	Name   string `db:"name"`
	Active bool   `db:"active"`
}

type AggregatorNode struct {
	Aggregator
	Raft *raft.Raft

	CollectedPrices map[int64][]int64
	AggregatorMutex sync.Mutex

	LastLocalAggregateTime time.Time
	RoundID                int64

	nodeCtx    context.Context
	nodeCancel context.CancelFunc
	isRunning  bool
}

type RoundSyncMessage struct {
	LeaderID string `json:"leaderID"`
	RoundID  int64  `json:"roundID"`
}

type PriceDataMessage struct {
	RoundID   int64 `json:"roundID"`
	PriceData int64 `json:"priceData"`
}
