package aggregator

import (
	"context"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/raft"
)

const (
	RoundSync        raft.MessageType = "roundSync"
	RoundReply       raft.MessageType = "roundReply"
	TriggerAggregate raft.MessageType = "triggerAggregate"
	PriceData        raft.MessageType = "priceData"

	SelectActiveAggregatorsQuery     = `SELECT * FROM aggregators WHERE active = true`
	SelectLatestLocalAggregateQuery  = `SELECT * FROM local_aggregates WHERE name = @name ORDER BY timestamp DESC LIMIT 1`
	InsertGlobalAggregateQuery       = `INSERT INTO global_aggregates (name, value, round) VALUES (@name, @value, @round) RETURNING *`
	SelectLatestGlobalAggregateQuery = `SELECT * FROM global_aggregates WHERE name = @name ORDER BY round DESC LIMIT 1`
)

type redisLocalAggregate struct {
	Value     int64     `json:"value"`
	Timestamp time.Time `json:"timestamp"`
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
}

type Aggregator struct {
	ID     int64  `db:"id"`
	Name   string `db:"name"`
	Active bool   `db:"active"`
}

type AggregatorNode struct {
	Aggregator
	Raft *raft.Raft

	LeaderJobTicker  *time.Ticker
	LeaderJobTimeout *time.Duration

	CollectedPrices map[int64][]int64
	AggregatorMutex sync.Mutex

	LastLocalAggregateTime time.Time
	RoundID                int64
	RoundSyncReplies       int

	nodeCtx    context.Context
	nodeCancel context.CancelFunc
	isRunning  bool
}

type RoundSyncMessage struct {
	LeaderID string `json:"leaderID"`
	RoundID  int64  `json:"roundID"`
}

type RoundReplyMessage struct {
	RoundId int64 `json:"roundId"`
}

type TriggerAggregateMessage struct {
	RoundID int64 `json:"roundID"`
}

type PriceDataMessage struct {
	RoundID   int64 `json:"roundID"`
	PriceData int64 `json:"priceData"`
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
