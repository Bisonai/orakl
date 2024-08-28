package aggregator

import (
	"context"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/bus"
	"bisonai.com/miko/node/pkg/chain/helper"
	"bisonai.com/miko/node/pkg/common/types"
	"bisonai.com/miko/node/pkg/raft"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

const (
	GLOBAL_AGGREGATE_ERR_THRESHOLD = 0.3
	AGREEMENT_QUORUM               = 0.5

	Trigger   raft.MessageType = "trigger"
	PriceData raft.MessageType = "priceData"
	PriceFix  raft.MessageType = "priceFix"
	ProofMsg  raft.MessageType = "proof"

	SelectConfigQuery                = `SELECT id, name, aggregate_interval FROM configs`
	SelectLatestLocalAggregateQuery  = `SELECT * FROM local_aggregates WHERE config_id = @config_id ORDER BY timestamp DESC LIMIT 1`
	InsertGlobalAggregateQuery       = `INSERT INTO global_aggregates (config_id, value, round, timestamp) VALUES (@config_id, @value, @round, @timestamp) RETURNING *`
	SelectLatestGlobalAggregateQuery = `SELECT * FROM global_aggregates WHERE config_id = @config_id ORDER BY round DESC LIMIT 1`
	InsertProofQuery                 = `INSERT INTO proofs (config_id, round, proof) VALUES (@config_id, @round, @proof) RETURNING *`
)

type LocalAggregate = types.LocalAggregate
type Proof = types.Proof
type GlobalAggregate = types.GlobalAggregate

type SubmissionData struct {
	GlobalAggregate GlobalAggregate
	Proof           Proof
}

type App struct {
	Bus                       *bus.MessageBus
	Aggregators               map[int32]*Aggregator
	GlobalAggregateBulkWriter *GlobalAggregateBulkWriter
	Host                      host.Host
	Pubsub                    *pubsub.PubSub
	Signer                    *helper.Signer
	LatestLocalAggregates     *LatestLocalAggregates
}

type Config struct {
	ID                int32  `db:"id"`
	Name              string `db:"name"`
	AggregateInterval int32  `db:"aggregate_interval"`
}

type RoundTriggers struct {
	locked map[int32]bool
	mu     sync.Mutex
}

func (r *RoundTriggers) leaveOnlyLast10Entries(roundID int32) {
	r.mu.Lock()
	defer r.mu.Unlock()

	newLocked := make(map[int32]bool)

	for i := roundID; i > roundID-10; i-- {
		if val, exists := r.locked[i]; exists {
			newLocked[i] = val
		}
	}

	r.locked = newLocked
}

type RoundPrices struct {
	senders map[int32][]string
	prices  map[int32][]int64
	locked  map[int32]bool
	mu      sync.Mutex
}

func (r *RoundPrices) isReplay(roundID int32, sender string) bool {
	for _, s := range r.senders[roundID] {
		if s == sender {
			return true
		}
	}
	return false
}

func (r *RoundPrices) leaveOnlyLast10Entries(roundID int32) {
	r.mu.Lock()
	defer r.mu.Unlock()

	newLocked := make(map[int32]bool)
	for i := roundID; i > roundID-10; i-- {
		if val, exists := r.locked[i]; exists {
			newLocked[i] = val
		}
	}
	r.locked = newLocked

	newPrices := make(map[int32][]int64)
	for i := roundID; i > roundID-10; i-- {
		if val, exists := r.prices[i]; exists {
			newPrices[i] = val
		}
	}
	r.prices = newPrices

	newSenders := make(map[int32][]string)
	for i := roundID; i > roundID-10; i-- {
		if val, exists := r.senders[i]; exists {
			newSenders[i] = val
		}
	}
	r.senders = newSenders
}

type RoundPriceFixes struct {
	locked map[int32]bool
	mu     sync.Mutex
}

func (r *RoundPriceFixes) leaveOnlyLast10Entries(roundID int32) {
	r.mu.Lock()
	defer r.mu.Unlock()

	newLocked := make(map[int32]bool)
	for i := roundID; i > roundID-10; i-- {
		if val, exists := r.locked[i]; exists {
			newLocked[i] = val
		}
	}
	r.locked = newLocked
}

type RoundProofs struct {
	senders map[int32][]string
	proofs  map[int32][][]byte
	locked  map[int32]bool
	mu      sync.Mutex
}

func (r *RoundProofs) isReplay(roundID int32, sender string) bool {
	for _, s := range r.senders[roundID] {
		if s == sender {
			return true
		}
	}
	return false
}

func (r *RoundProofs) leaveOnlyLast10Entries(roundID int32) {
	r.mu.Lock()
	defer r.mu.Unlock()

	newLocked := make(map[int32]bool)
	for i := roundID; i > roundID-10; i-- {
		if val, exists := r.locked[i]; exists {
			newLocked[i] = val
		}
	}
	r.locked = newLocked

	newProofs := make(map[int32][][]byte)
	for i := roundID; i > roundID-10; i-- {
		if val, exists := r.proofs[i]; exists {
			newProofs[i] = val
		}
	}
	r.proofs = newProofs

	newSenders := make(map[int32][]string)
	for i := roundID; i > roundID-10; i-- {
		if val, exists := r.senders[i]; exists {
			newSenders[i] = val
		}
	}
	r.senders = newSenders
}

type Aggregator struct {
	Config
	Raft *raft.Raft

	LatestLocalAggregates *LatestLocalAggregates
	roundLocalAggregate   map[int32]int64

	roundTriggers   *RoundTriggers
	roundPrices     *RoundPrices
	roundPriceFixes *RoundPriceFixes
	roundProofs     *RoundProofs

	RoundID int32
	Signer  *helper.Signer

	nodeCtx    context.Context
	nodeCancel context.CancelFunc
	isRunning  bool

	mu  sync.RWMutex
	bus *bus.MessageBus
}

type PriceDataMessage struct {
	RoundID   int32     `json:"roundID"`
	PriceData int64     `json:"priceData"`
	Timestamp time.Time `json:"timestamp"`
}

type PriceFixMessage struct {
	RoundID   int32     `json:"roundID"`
	PriceData int64     `json:"priceData"`
	Timestamp time.Time `json:"timestamp"`
}

type ProofMessage struct {
	RoundID   int32     `json:"roundID"`
	Value     int64     `json:"value"`
	Proof     []byte    `json:"proof"`
	Timestamp time.Time `json:"timestamp"`
}

type TriggerMessage struct {
	LeaderID  string    `json:"leaderID"`
	RoundID   int32     `json:"roundID"`
	Timestamp time.Time `json:"timestamp"`
}

type LatestLocalAggregates struct {
	LocalAggregateMap map[int32]*LocalAggregate
	mu                sync.RWMutex
}

func NewLatestLocalAggregates() *LatestLocalAggregates {
	return &LatestLocalAggregates{
		LocalAggregateMap: map[int32]*LocalAggregate{},
	}
}

func (a *LatestLocalAggregates) Load(id int32) (*LocalAggregate, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result, ok := a.LocalAggregateMap[id]
	return result, ok
}

func (a *LatestLocalAggregates) Store(id int32, aggregate *LocalAggregate) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.LocalAggregateMap[id] = aggregate
}
