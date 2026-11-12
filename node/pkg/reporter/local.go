package reporter

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"bisonai.com/miko/node/pkg/chain/helper"
	"bisonai.com/miko/node/pkg/chain/proof"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
	"github.com/rs/zerolog/log"
)

const (
	// LocalMaxAge drops in-node data older than this: the node stopped
	// producing rounds for the pair, so it has nothing current to submit.
	LocalMaxAge = 60 * time.Second

	whitelistRefreshInterval = 10 * time.Second
	// whitelistRefreshCooldown bounds refreshes triggered by rejected proofs.
	whitelistRefreshCooldown = time.Second
)

// localSource feeds a reporter running inside the node: submission data comes
// from the node's own aggregators (Ingest), and only pairs whose latest round
// this node led are submitted.
type localSource struct {
	leaders sync.Map // map[symbol]bool, from the latest ingested round

	chainHelper     *helper.ChainHelper
	contractAddress string

	whitelistMu sync.RWMutex
	whitelist   []common.Address

	lastRefresh atomic.Int64 // UnixNano of the last on-demand refresh

	now func() time.Time
}

func newLocalSource() *localSource {
	return &localSource{now: time.Now}
}

// NewNodeApp returns a reporter that runs inside the node: it gets data
// through Ingest instead of DAL and submits only pairs this node leads.
func NewNodeApp() *App {
	a := New()
	a.local = newLocalSource()
	return a
}

// Ingest stores a global aggregate produced by this node; leader is whether
// this node led the round. Older data than already stored is ignored.
func (a *App) Ingest(symbol string, value int64, timestamp time.Time, rawProof []byte, leader bool) {
	if a.local == nil {
		return
	}
	data := SubmissionData{
		Symbol:        symbol,
		Value:         value,
		AggregateTime: timestamp.UnixMilli(),
		Proof:         rawProof,
		FeedHash:      [32]byte(crypto.Keccak256([]byte(symbol))),
	}

	a.latestMu.Lock()
	defer a.latestMu.Unlock()
	if prev, ok := a.LatestDataMap.Load(symbol); ok {
		if old, ok := prev.(SubmissionData); ok && old.AggregateTime > data.AggregateTime {
			return
		}
	}
	a.LatestDataMap.Store(symbol, data)
	a.local.leaders.Store(symbol, leader)
}

func (l *localSource) setWhitelist(whitelist []common.Address) {
	l.whitelistMu.Lock()
	defer l.whitelistMu.Unlock()
	l.whitelist = whitelist
}

func (l *localSource) getWhitelist() []common.Address {
	l.whitelistMu.RLock()
	defer l.whitelistMu.RUnlock()
	return append([]common.Address(nil), l.whitelist...)
}

func (l *localSource) refreshWhitelist(ctx context.Context) {
	whitelist, err := ReadOnchainWhitelist(ctx, l.chainHelper, l.contractAddress, GET_ONCHAIN_WHITELIST)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to refresh whitelist")
		return
	}
	l.setWhitelist(whitelist)
}

func (l *localSource) runWhitelistRefresh(ctx context.Context) {
	ticker := time.NewTicker(whitelistRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.refreshWhitelist(ctx)
		}
	}
}

// maybeRefreshWhitelist refreshes at once after a proof was rejected for a
// non-whitelisted signer (signer rotation), at most once per cooldown.
func (l *localSource) maybeRefreshWhitelist(ctx context.Context) {
	if l.chainHelper == nil {
		return
	}
	now := l.now().UnixNano()
	last := l.lastRefresh.Load()
	if now-last < int64(whitelistRefreshCooldown) || !l.lastRefresh.CompareAndSwap(last, now) {
		return
	}
	go l.refreshWhitelist(ctx)
}

func (l *localSource) latest(latestDataMap *sync.Map, pairs []string) map[string]SubmissionData {
	result := make(map[string]SubmissionData, len(pairs))
	for _, pair := range pairs {
		if data, ok := GetLatestData(latestDataMap, pair); ok {
			result[pair] = data
		}
	}
	return result
}

// prepare keeps the pairs this node led in their latest round with data
// newer than LocalMaxAge, and orders their proofs for SubmissionProxy.
func (l *localSource) prepare(ctx context.Context, pairs map[string]SubmissionData) map[string]SubmissionData {
	whitelist := l.getWhitelist()
	cutoff := l.now().Add(-LocalMaxAge).UnixMilli()

	result := make(map[string]SubmissionData, len(pairs))
	for pair, data := range pairs {
		if leader, ok := l.leaders.Load(pair); !ok || !leader.(bool) {
			continue
		}
		if data.AggregateTime < cutoff {
			continue
		}
		ordered, err := proof.Order(data.Proof, data.Value, time.UnixMilli(data.AggregateTime), pair, whitelist)
		if err != nil {
			log.Error().Str("Player", "Reporter").Str("pair", pair).Err(err).Msg("failed to order proof")
			if errors.Is(err, errorSentinel.ErrDalSignerNotWhitelisted) {
				l.maybeRefreshWhitelist(ctx)
			}
			continue
		}
		data.Proof = ordered
		result[pair] = data
	}
	return result
}
