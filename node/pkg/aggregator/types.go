package aggregator

import (
	"context"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/common"
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

	SelectConfigQuery                = `SELECT id, name, aggregate_interval FROM configs`
	SelectLatestLocalAggregateQuery  = `SELECT * FROM local_aggregates WHERE config_id = @config_id ORDER BY timestamp DESC LIMIT 1`
	InsertGlobalAggregateQuery       = `INSERT INTO global_aggregates (config_id, value, round, timestamp) VALUES (@config_id, @value, @round, @timestamp) RETURNING *`
	SelectLatestGlobalAggregateQuery = `SELECT * FROM global_aggregates WHERE config_id = @config_id ORDER BY round DESC LIMIT 1`
	InsertProofQuery                 = `INSERT INTO proofs (config_id, round, proof) VALUES (@config_id, @round, @proof) RETURNING *`
)

type LocalAggregate common.LocalAggregate

type PgsqlProof struct {
	ID        int64     `db:"id"`
	ConfigID  int32     `json:"configId"`
	Round     int32     `json:"round"`
	Proof     []byte    `json:"proof"`
	Timestamp time.Time `json:"timestamp"`
}

type Proof struct {
	ConfigID int32  `json:"configId"`
	Round    int32  `json:"round"`
	Proof    []byte `json:"proofs"`
}

type GlobalAggregate common.GlobalAggregate

type App struct {
	Bus         *bus.MessageBus
	Aggregators map[int32]*Aggregator
	Host        host.Host
	Pubsub      *pubsub.PubSub
}

type Config struct {
	ID                int32  `db:"id"`
	Name              string `db:"name"`
	AggregateInterval int32  `db:"aggregate_interval"`
}

type Aggregator struct {
	Config
	Raft *raft.Raft

	CollectedPrices          map[int32][]int64
	CollectedProofs          map[int32][][]byte
	CollectedAgreements      map[int32][]bool
	PreparedLocalAggregates  map[int32]int64
	PreparedGlobalAggregates map[int32]GlobalAggregate
	SyncedTimes              map[int32]time.Time
	AggregatorMutex          sync.Mutex

	RoundID int32

	SignHelper *helper.SignHelper

	nodeCtx    context.Context
	nodeCancel context.CancelFunc
	isRunning  bool
}

type RoundSyncMessage struct {
	LeaderID  string    `json:"leaderID"`
	RoundID   int32     `json:"roundID"`
	Timestamp time.Time `json:"timestamp"`
}

type PriceDataMessage struct {
	RoundID   int32 `json:"roundID"`
	PriceData int64 `json:"priceData"`
}

type ProofMessage struct {
	RoundID int32  `json:"roundID"`
	Proof   []byte `json:"proof"`
}

type SyncReplyMessage struct {
	RoundID int32 `json:"roundID"`
	Agreed  bool  `json:"agreed"`
}

type TriggerMessage struct {
	LeaderID string `json:"leaderID"`
	RoundID  int32  `json:"roundID"`
}
