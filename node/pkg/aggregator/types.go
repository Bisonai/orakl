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

	// roundHistory is how many rounds up to an accepted trigger's round keep
	// their state when it prunes.
	roundHistory = 10
	// roundLookahead is how many rounds ahead of the current one keep their
	// state when a trigger prunes (messages racing ahead of their trigger).
	roundLookahead = 2

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
	// CanLead, if set before Run, gates every feed's leadership on this
	// node's health (raft.Raft.CanLead).
	CanLead func() bool

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

// pruneOutside keeps only rounds in [low, high].
func (r *RoundTriggers) pruneOutside(low, high int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.locked = keepRounds(r.locked, low, high)
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

// pruneOutside keeps only rounds in [low, high].
func (r *RoundPrices) pruneOutside(low, high int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.locked = keepRounds(r.locked, low, high)
	r.prices = keepRounds(r.prices, low, high)
	r.senders = keepRounds(r.senders, low, high)
}

type RoundPriceFixes struct {
	locked map[int32]bool
	mu     sync.Mutex
}

// pruneOutside keeps only rounds in [low, high].
func (r *RoundPriceFixes) pruneOutside(low, high int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.locked = keepRounds(r.locked, low, high)
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

// pruneOutside keeps only rounds in [low, high].
func (r *RoundProofs) pruneOutside(low, high int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.locked = keepRounds(r.locked, low, high)
	r.proofs = keepRounds(r.proofs, low, high)
	r.senders = keepRounds(r.senders, low, high)
}

// keepRounds returns a new map with only the rounds in [low, high]; a new map
// so memory of a burst of rounds is released (Go maps don't shrink).
func keepRounds[V any](m map[int32]V, low, high int64) map[int32]V {
	kept := make(map[int32]V)
	for roundID, v := range m {
		if r := int64(roundID); r >= low && r <= high {
			kept[roundID] = v
		}
	}
	return kept
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

	// peerRoundID is the highest round a peer reported using while it was
	// ahead of this node as leader; this node's rounds up to it are stale and
	// produce no aggregate (under mu)
	peerRoundID int32

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
	// LatestRoundID is the sender's RoundID when it answered the trigger, so a
	// lagging leader can move past rounds its peers already used (#2867).
	// 0: not reported (older nodes).
	LatestRoundID int32 `json:"latestRoundID,omitempty"`
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
