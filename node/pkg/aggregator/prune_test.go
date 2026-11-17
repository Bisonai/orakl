package aggregator

import (
	"context"
	"maps"
	"slices"
	"testing"
	"time"

	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/raft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedRounds fills every per-round map for each given round.
func seedRounds(agg *Aggregator, rounds ...int32) {
	for _, r := range rounds {
		agg.roundTriggers.locked[r] = true
		agg.roundPrices.locked[r] = true
		agg.roundPrices.prices[r] = []int64{int64(r)}
		agg.roundPrices.senders[r] = []string{"peer"}
		agg.roundPriceFixes.locked[r] = true
		agg.roundProofs.locked[r] = true
		agg.roundProofs.proofs[r] = [][]byte{{byte(r)}}
		agg.roundProofs.senders[r] = []string{"peer"}
	}
}

func sortedKeys[V any](m map[int32]V) []int32 {
	return slices.Sorted(maps.Keys(m))
}

// assertRounds checks that every per-round map holds exactly want.
func assertRounds(t *testing.T, agg *Aggregator, want []int32) {
	t.Helper()
	assert.Equal(t, want, sortedKeys(agg.roundTriggers.locked), "roundTriggers.locked")
	assert.Equal(t, want, sortedKeys(agg.roundPrices.locked), "roundPrices.locked")
	assert.Equal(t, want, sortedKeys(agg.roundPrices.prices), "roundPrices.prices")
	assert.Equal(t, want, sortedKeys(agg.roundPrices.senders), "roundPrices.senders")
	assert.Equal(t, want, sortedKeys(agg.roundPriceFixes.locked), "roundPriceFixes.locked")
	assert.Equal(t, want, sortedKeys(agg.roundProofs.locked), "roundProofs.locked")
	assert.Equal(t, want, sortedKeys(agg.roundProofs.proofs), "roundProofs.proofs")
	assert.Equal(t, want, sortedKeys(agg.roundProofs.senders), "roundProofs.senders")
}

func roundRange(from, to int32) []int32 {
	var out []int32
	for r := from; r <= to; r++ {
		out = append(out, r)
	}
	return out
}

func sendTrigger(t *testing.T, ctx context.Context, agg *Aggregator, from string, round int32) error {
	t.Helper()
	data, err := encodeInner(TriggerMessage{LeaderID: from, RoundID: round, Timestamp: time.Now()})
	require.NoError(t, err)
	return agg.HandleTriggerMessage(ctx, raft.Message{Type: Trigger, SentFrom: from, Data: data})
}

// Rejected triggers (non-leader with low/high RoundID, RoundID 0) must not
// touch round state (#2853).
func TestRejectedTriggerKeepsRoundState(t *testing.T) {
	ctx := context.Background()
	agg := newRoundTestAggregator(t, ctx, "PRUNE-REJECT", nil)
	agg.RoundID = 50
	agg.Raft.LeaderID = "leader-peer"
	seedRounds(agg, roundRange(41, 50)...)

	err := sendTrigger(t, ctx, agg, "rogue-peer", 5)
	assert.ErrorIs(t, err, errorSentinel.ErrAggregatorNonLeaderRaftMessage)
	assertRounds(t, agg, roundRange(41, 50))

	err = sendTrigger(t, ctx, agg, "rogue-peer", 1000)
	assert.ErrorIs(t, err, errorSentinel.ErrAggregatorNonLeaderRaftMessage)
	assertRounds(t, agg, roundRange(41, 50))

	err = sendTrigger(t, ctx, agg, "leader-peer", 0)
	assert.ErrorIs(t, err, errorSentinel.ErrAggregatorInvalidRaftMessage)
	assertRounds(t, agg, roundRange(41, 50))

	assert.Equal(t, int32(50), agg.RoundID)
}

// An accepted leader trigger keeps roundHistory rounds up to its round plus
// roundLookahead rounds ahead of the current one; a stale leader trigger never
// drops current rounds.
func TestLeaderTriggerPrunesOnlyOldRounds(t *testing.T) {
	ctx := context.Background()
	agg := newRoundTestAggregator(t, ctx, "PRUNE-LEADER", nil)
	agg.RoundID = 50
	agg.Raft.LeaderID = "leader-peer"
	// 52, 53: state racing ahead of its trigger (kept); 54, 60: too far ahead (dropped)
	seedRounds(agg, append(roundRange(35, 50), 52, 53, 54, 60)...)
	kept := append(roundRange(42, 50), 52, 53)
	keptTriggers := roundRange(42, 53) // 51 locked by the trigger itself

	require.NoError(t, sendTrigger(t, ctx, agg, "leader-peer", 51))
	assert.Equal(t, int32(51), agg.RoundID)
	assert.Equal(t, keptTriggers, sortedKeys(agg.roundTriggers.locked))
	assert.Equal(t, kept, sortedKeys(agg.roundPrices.prices))
	assert.Equal(t, kept, sortedKeys(agg.roundPrices.senders))
	assert.Equal(t, kept, sortedKeys(agg.roundProofs.proofs))
	assert.Equal(t, kept, sortedKeys(agg.roundPriceFixes.locked))

	// a stale leader trigger keeps current rounds and its own round's lock
	// (so a replay of it is deduplicated)
	require.NoError(t, sendTrigger(t, ctx, agg, "leader-peer", 30))
	assert.Equal(t, int32(51), agg.RoundID)
	assert.Equal(t, append([]int32{30}, keptTriggers...), sortedKeys(agg.roundTriggers.locked))
	assert.Equal(t, kept, sortedKeys(agg.roundPrices.prices))
	assert.Equal(t, kept, sortedKeys(agg.roundProofs.proofs))

	// the next leader trigger prunes it again
	require.NoError(t, sendTrigger(t, ctx, agg, "leader-peer", 52))
	assert.Equal(t, roundRange(43, 53), sortedKeys(agg.roundTriggers.locked))
	assert.Equal(t, append(roundRange(43, 50), 52, 53), sortedKeys(agg.roundPrices.prices))
}
