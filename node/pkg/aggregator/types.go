package aggregator

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"bisonai.com/miko/node/pkg/bus"
	"bisonai.com/miko/node/pkg/chain/helper"
	"bisonai.com/miko/node/pkg/common/types"
	"bisonai.com/miko/node/pkg/nodeconfig"
	"bisonai.com/miko/node/pkg/nodestate"
	"bisonai.com/miko/node/pkg/raft"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
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

	// ConfigureAggregator, if set before Run, is called with each aggregator
	// after it is built and before it starts (tests: raft timers, clock).
	ConfigureAggregator func(*Aggregator)

	// rounds persists RoundID advances of all aggregators to State (nil
	// without State).
	rounds *roundPersister

	// rafts maps each feed name to its current aggregator's Raft (Leads)
	rafts sync.Map // map[string]*raft.Raft

	// running aggregator Run goroutines (WaitStopped)
	runs runTracker
}

type Config = nodeconfig.Config

type RoundTriggers struct {
	// locked holds, per round, the trigger the round was answered for
	locked map[int32]roundTrigger
	mu     sync.Mutex
}

// roundTrigger identifies a trigger: a duplicate delivery of it has the same
// sender and timestamp, a leader reusing the round ID (fresh or lost round
// state) has another sender or timestamp (#2927).
type roundTrigger struct {
	sender    string
	timestamp time.Time
}

func (t roundTrigger) same(sender string, timestamp time.Time) bool {
	return t.sender == sender && t.timestamp.Equal(timestamp)
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

	// Now, if set before Run, replaces time.Now for the timestamp of the
	// rounds this node triggers as leader (tests: clock offset).
	Now func() time.Time

	// rounds persists every RoundID advance (nil: not persisted)
	rounds *roundPersister

	// peerRoundID is the highest round a peer reported using while it was
	// ahead of this node as leader; this node's rounds up to it are stale and
	// produce no aggregate (under mu)
	peerRoundID int32

	// onSubmission gets each round's submission data (nil: none)
	onSubmission SubmissionHook

	// noLocalAggregate: the last trigger found no fresh local aggregate;
	// noPrices: the last round as leader collected no price (both log only
	// the transitions)
	noLocalAggregate atomic.Bool
	noPrices         atomic.Bool

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

// DefaultLocalAggregateMaxAge is how old this node's local aggregate may be
// and still be voted. The fetcher republishes every config every
// LOCAL_AGGREGATE_INTERVAL (200ms) while its feeds are fresh (each feed within
// the config's feedDataFreshness) and stops publishing once they are not, so
// an older aggregate means the fetcher stopped; voting it would re-submit a
// frozen price under fresh round timestamps (#2964). 60s is the fetcher's own
// staleness window (feedDataFreshness of every config) and spans a fetcher
// REFRESH, which restarts the local aggregators one by one 50-150ms apart
// (~25s for 235 configs). LOCAL_AGGREGATE_MAX_AGE overrides it; without it, a
// LOCAL_AGGREGATE_INTERVAL above 12s raises it to
// localAggregateMaxAgeIntervals publish intervals.
const DefaultLocalAggregateMaxAge = 60 * time.Second

// localAggregateMaxAgeIntervals is the least number of publish intervals the
// default max age spans.
const localAggregateMaxAgeIntervals = 5

type LatestLocalAggregates struct {
	LocalAggregateMap map[int32]*LocalAggregate
	// MaxAge: LoadFresh treats an older aggregate as missing. Set it before
	// the aggregators run.
	MaxAge time.Duration
	mu     sync.RWMutex
}

func NewLatestLocalAggregates() *LatestLocalAggregates {
	return &LatestLocalAggregates{
		LocalAggregateMap: map[int32]*LocalAggregate{},
		MaxAge:            localAggregateMaxAge(),
	}
}

func localAggregateMaxAge() time.Duration {
	// the fetcher's publish interval (its env and default
	// fetcher.DefaultLocalAggregateInterval, fetcher.LocalAggregator.Run)
	interval, err := time.ParseDuration(os.Getenv("LOCAL_AGGREGATE_INTERVAL"))
	if err != nil || interval <= 0 {
		interval = 200 * time.Millisecond
	}
	def := max(DefaultLocalAggregateMaxAge, localAggregateMaxAgeIntervals*interval)

	raw := os.Getenv("LOCAL_AGGREGATE_MAX_AGE")
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		log.Warn().Str("Player", "Aggregator").Str("LOCAL_AGGREGATE_MAX_AGE", raw).Dur("default", def).Msg("invalid local aggregate max age, using default")
		return def
	}
	if d < 2*interval {
		log.Warn().Str("Player", "Aggregator").Dur("LOCAL_AGGREGATE_MAX_AGE", d).Dur("LOCAL_AGGREGATE_INTERVAL", interval).Msg("local aggregate max age below two publish intervals: local aggregates will often be voted as no price")
	}
	return d
}

func (a *LatestLocalAggregates) Load(id int32) (*LocalAggregate, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result, ok := a.LocalAggregateMap[id]
	return result, ok
}

// LoadFresh is Load, but an aggregate more than MaxAge older (or newer: a
// skewed stamp) than now is missing.
func (a *LatestLocalAggregates) LoadFresh(id int32, now time.Time) (*LocalAggregate, bool) {
	result, ok := a.Load(id)
	if !ok {
		return nil, false
	}
	maxAge := a.MaxAge
	if maxAge <= 0 { // not built by NewLatestLocalAggregates
		maxAge = DefaultLocalAggregateMaxAge
	}
	if age := now.Sub(result.Timestamp); age > maxAge || age < -maxAge {
		return nil, false
	}
	return result, true
}

// Store keeps the newest aggregate per config: every bus message is handled
// on its own goroutine, so an older one may arrive last. A cached aggregate
// stamped in the future is replaced, so it can't pin the value.
func (a *LatestLocalAggregates) Store(id int32, aggregate *LocalAggregate) {
	if aggregate == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if prev, ok := a.LocalAggregateMap[id]; ok && prev.Timestamp.After(aggregate.Timestamp) && !prev.Timestamp.After(time.Now()) {
		return
	}
	a.LocalAggregateMap[id] = aggregate
}

// Retain drops the aggregates of configs not in configs (removed by a sync).
func (a *LatestLocalAggregates) Retain(configs []Config) {
	keep := make(map[int32]bool, len(configs))
	for _, c := range configs {
		keep[c.ID] = true
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for id := range a.LocalAggregateMap {
		if !keep[id] {
			delete(a.LocalAggregateMap, id)
		}
	}
}
