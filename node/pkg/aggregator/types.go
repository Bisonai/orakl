package aggregator

import (
	"context"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/raft"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

const (
	AGREEMENT_QUORUM = 0.5

	RoundSync raft.MessageType = "roundSync"
	SyncReply raft.MessageType = "syncReply"
	Trigger   raft.MessageType = "trigger"
	PriceData raft.MessageType = "priceData"
	ProofMsg  raft.MessageType = "proof"

	SelectActiveAggregatorsQuery     = `SELECT * FROM aggregators WHERE active = true`
	SelectLatestLocalAggregateQuery  = `SELECT * FROM local_aggregates WHERE name = @name ORDER BY timestamp DESC LIMIT 1`
	InsertGlobalAggregateQuery       = `INSERT INTO global_aggregates (name, value, round, timestamp) VALUES (@name, @value, @round, @timestamp) RETURNING *`
	SelectLatestGlobalAggregateQuery = `SELECT * FROM global_aggregates WHERE name = @name ORDER BY round DESC LIMIT 1`
	InsertProofQuery                 = `INSERT INTO proofs (name, round, proof) VALUES (@name, @round, @proof) RETURNING *`
)

type LocalAggregate struct {
	Value     int64     `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

type PgsqlProof struct {
	ID        int64     `db:"id"`
	Name      string    `json:"name"`
	Round     int64     `json:"round"`
	Proof     []byte    `json:"proof"`
	Timestamp time.Time `json:"timestamp"`
}

type Proof struct {
	Name  string `json:"name"`
	Round int64  `json:"round"`
	Proof []byte `json:"proofs"`
}

type PgsLocalAggregate struct {
	Name      string    `db:"name"`
	Value     int64     `db:"value"`
	Timestamp time.Time `db:"timestamp"`
}

type GlobalAggregate struct {
	Name      string    `db:"name" json:"name"`
	Value     int64     `db:"value" json:"value"`
	Round     int64     `db:"round" json:"round"`
	Timestamp time.Time `db:"timestamp" json:"timestamp"`
}

type App struct {
	Bus         *bus.MessageBus
	Aggregators map[int64]*Aggregator
	Host        host.Host
	Pubsub      *pubsub.PubSub
}

type AggregatorModel struct {
	ID       int64  `db:"id"`
	Name     string `db:"name"`
	Active   bool   `db:"active"`
	Interval int    `db:"interval"`
}

type Aggregator struct {
	AggregatorModel
	Raft *raft.Raft

	CollectedPrices         map[int64][]int64
	CollectedProofs         map[int64][][]byte
	CollectedAgreements     map[int64][]bool
	PreparedLocalAggregates map[int64]int64
	SyncedTimes             map[int64]time.Time
	AggregatorMutex         sync.Mutex

	RoundID int64

	SignHelper *helper.SignHelper

	nodeCtx    context.Context
	nodeCancel context.CancelFunc
	isRunning  bool
}

type RoundSyncMessage struct {
	LeaderID  string    `json:"leaderID"`
	RoundID   int64     `json:"roundID"`
	Timestamp time.Time `json:"timestamp"`
}

type PriceDataMessage struct {
	RoundID   int64 `json:"roundID"`
	PriceData int64 `json:"priceData"`
}

type ProofMessage struct {
	RoundID int64  `json:"roundID"`
	Proof   []byte `json:"proof"`
}

type SyncReplyMessage struct {
	RoundID int64 `json:"roundID"`
	Agreed  bool  `json:"agreed"`
}

type TriggerMessage struct {
	LeaderID string `json:"leaderID"`
	RoundID  int64  `json:"roundID"`
}
