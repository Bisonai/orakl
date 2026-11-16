package aggregator

import (
	"context"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/bus"
	"bisonai.com/miko/node/pkg/chain/helper"
	"bisonai.com/miko/node/pkg/common/types"
	"bisonai.com/miko/node/pkg/nodeconfig"
	"bisonai.com/miko/node/pkg/nodestate"
	"bisonai.com/miko/node/pkg/raft"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

const (
	AGREEMENT_QUORUM = 0.5

	// roundHistory is how many rounds up to the current one keep their state.
	roundHistory = 10

	Trigger   raft.MessageType = "trigger"
	PriceData raft.MessageType = "priceData"
	PriceFix  raft.MessageType = "priceFix"
	ProofMsg  raft.MessageType = "proof"
)

type LocalAggregate = types.LocalAggregate
type Proof = types.Proof
type GlobalAggregate = types.GlobalAggregate

type SubmissionData struct {
	Symbol          string
	GlobalAggregate GlobalAggregate
	Proof           Proof
}

// SubmissionHook receives every global aggregate with its proof; leader is
// whether this node led the feed's round. It must not block.
type SubmissionHook func(data SubmissionData, leader bool)

type App struct {
	Bus                   *bus.MessageBus
	Configs               *nodeconfig.Store
	Aggregators           map[int32]*Aggregator
	Host                  host.Host
	Pubsub                *pubsub.PubSub
	Signer                *helper.Signer
	LatestLocalAggregates *LatestLocalAggregates
	State                 *nodestate.Store

	// OnSubmission, if set before Run, gets every aggregator's submission
	// data (in-node reporter).
	OnSubmission SubmissionHook

	// rounds persists RoundID advances of all aggregators to State (nil
	// without State).
	rounds *roundPersister

	// running aggregator Run goroutines (WaitStopped)
	runs runTracker
}

type Config = nodeconfig.Config

type RoundTriggers struct {
	locked map[int32]bool
	mu     sync.Mutex
}

// pruneBefore drops rounds older than cutoff and keeps every round >= cutoff.
func (r *RoundTriggers) pruneBefore(cutoff int32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	deleteRoundsBefore(r.locked, cutoff)
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

// pruneBefore drops rounds older than cutoff and keeps every round >= cutoff.
func (r *RoundPrices) pruneBefore(cutoff int32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	deleteRoundsBefore(r.locked, cutoff)
	deleteRoundsBefore(r.prices, cutoff)
	deleteRoundsBefore(r.senders, cutoff)
}

type RoundPriceFixes struct {
	locked map[int32]bool
	mu     sync.Mutex
}

// pruneBefore drops rounds older than cutoff and keeps every round >= cutoff.
func (r *RoundPriceFixes) pruneBefore(cutoff int32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	deleteRoundsBefore(r.locked, cutoff)
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

// pruneBefore drops rounds older than cutoff and keeps every round >= cutoff.
func (r *RoundProofs) pruneBefore(cutoff int32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	deleteRoundsBefore(r.locked, cutoff)
	deleteRoundsBefore(r.proofs, cutoff)
	deleteRoundsBefore(r.senders, cutoff)
}

// deleteRoundsBefore removes the entries of rounds older than cutoff.
func deleteRoundsBefore[V any](m map[int32]V, cutoff int32) {
	for roundID := range m {
		if roundID < cutoff {
			delete(m, roundID)
		}
	}
}

type Aggregator struct {
	Config
	Raft *raft.Raft

	LatestLocalAggregates *LatestLocalAggregates
	roundTriggers         *RoundTriggers
	roundPrices           *RoundPrices
	roundPriceFixes       *RoundPriceFixes
	roundProofs           *RoundProofs

	RoundID int32
	Signer  *helper.Signer

	// rounds persists every RoundID advance (nil: not persisted)
	rounds *roundPersister

	// onSubmission gets each round's submission data (nil: none)
	onSubmission SubmissionHook

	nodeCtx    context.Context
	nodeCancel context.CancelFunc
	isRunning  bool
	// done closes when the Run goroutine started by startAggregator returns
	done chan struct{}

	mu sync.RWMutex
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
