//nolint:all
package fetcher

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/bus"
	"github.com/stretchr/testify/assert"
)

func TestLocalAggregateValueMap_SetGet(t *testing.T) {
	m := NewLocalAggregateValueMap()
	_, ok := m.Get("missing")
	assert.False(t, ok, "missing key should report not present")

	m.Set("KRW-USD", 1474.4)
	entry, ok := m.Get("KRW-USD")
	assert.True(t, ok)
	assert.Equal(t, 1474.4, entry.Value)
	assert.WithinDuration(t, time.Now(), entry.Timestamp, time.Second)
}

func newLocalAggregatorForMultiply(t *testing.T, multiplyBy *string, freshnessMs *int) (*LocalAggregator, chan *LocalAggregate) {
	return newLocalAggregatorForMultiplyOp(t, multiplyBy, freshnessMs, false)
}

func newLocalAggregatorForMultiplyOp(t *testing.T, multiplyBy *string, freshnessMs *int, reciprocal bool) (*LocalAggregator, chan *LocalAggregate) {
	t.Helper()
	ch := make(chan *LocalAggregate, 1)
	mb := bus.New(10)
	// streamLocalAggregate publishes to the AGGREGATOR channel and
	// returns ErrBusChannelNotFound otherwise; subscribe a no-op consumer
	// to keep the publish path happy in tests.
	mb.Subscribe(bus.AGGREGATOR)
	return &LocalAggregator{
		Config: Config{
			ID:                   42,
			Name:                 "STG-KRW",
			FetchInterval:        2000,
			Decimals:             intPtr(8),
			FeedDataFreshness:    freshnessMs,
			MultiplyBy:           multiplyBy,
			MultiplyByReciprocal: reciprocal,
		},
		bus:                    mb,
		localAggregatesChannel: ch,
		localAggregateValueMap: NewLocalAggregateValueMap(),
	}, ch
}

func TestStreamLocalAggregate_NoMultiplyBy_PassesThrough(t *testing.T) {
	la, ch := newLocalAggregatorForMultiply(t, nil, intPtr(60_000))

	err := la.streamLocalAggregate(context.Background(), 0.218)
	assert.NoError(t, err)

	select {
	case agg := <-ch:
		// 0.218 * 10^8 = 21800000
		assert.Equal(t, int64(21_800_000), agg.Value)
		assert.Equal(t, int32(42), agg.ConfigID)
	case <-time.After(time.Second):
		t.Fatal("expected aggregate on channel")
	}
}

func TestStreamLocalAggregate_MultiplyBy_AppliesMultiplier(t *testing.T) {
	mb := "KRW-USD"
	la, ch := newLocalAggregatorForMultiply(t, &mb, intPtr(60_000))
	la.localAggregateValueMap.Set("KRW-USD", 1474.4)

	err := la.streamLocalAggregate(context.Background(), 0.218)
	assert.NoError(t, err)

	select {
	case agg := <-ch:
		// 0.218 * 1474.4 = 321.4192, * 10^8 = 32_141_920_000
		assert.InDelta(t, int64(32_141_920_000), agg.Value, 100, "STG-USDT * KRW-USD should equal STG-KRW")
	case <-time.After(time.Second):
		t.Fatal("expected aggregate on channel")
	}
}

func TestStreamLocalAggregate_MultiplyBy_MissingSource_Skips(t *testing.T) {
	mb := "KRW-USD"
	la, ch := newLocalAggregatorForMultiply(t, &mb, intPtr(60_000))
	// Don't set KRW-USD in the map

	err := la.streamLocalAggregate(context.Background(), 0.218)
	assert.NoError(t, err)

	select {
	case <-ch:
		t.Fatal("should not emit when multiplier source is unavailable")
	case <-time.After(50 * time.Millisecond):
		// expected: no emit
	}
}

func TestStreamLocalAggregate_MultiplyBy_StaleSource_Skips(t *testing.T) {
	mb := "KRW-USD"
	la, ch := newLocalAggregatorForMultiply(t, &mb, intPtr(60_000))
	// Force a stale timestamp directly into the map.
	la.localAggregateValueMap.Mu.Lock()
	la.localAggregateValueMap.Data["KRW-USD"] = LocalAggregateValueEntry{
		Value:     1474.4,
		Timestamp: time.Now().Add(-10 * time.Minute),
	}
	la.localAggregateValueMap.Mu.Unlock()

	err := la.streamLocalAggregate(context.Background(), 0.218)
	assert.NoError(t, err)

	select {
	case <-ch:
		t.Fatal("should not emit when multiplier source is stale")
	case <-time.After(50 * time.Millisecond):
		// expected: no emit
	}
}

func TestStreamLocalAggregate_CachesOwnRawValue(t *testing.T) {
	la, _ := newLocalAggregatorForMultiply(t, nil, intPtr(60_000))

	err := la.streamLocalAggregate(context.Background(), 0.218)
	assert.NoError(t, err)

	entry, ok := la.localAggregateValueMap.Get("STG-KRW")
	assert.True(t, ok, "config should cache its own raw value for downstream synthetic configs")
	assert.Equal(t, 0.218, entry.Value)
}

func TestStreamLocalAggregate_MultiplyByReciprocal_DividesBySource(t *testing.T) {
	mb := "KRW-USD"
	la, ch := newLocalAggregatorForMultiplyOp(t, &mb, intPtr(60_000), true)
	// KRW-USD's raw value intentionally stores USD-per-KRW (the reciprocal
	// of the natural KRW-per-USD rate); the synthetic config divides by it
	// to get back to KRW-per-USD scale.
	la.localAggregateValueMap.Set("KRW-USD", 0.000678)

	err := la.streamLocalAggregate(context.Background(), 0.218)
	assert.NoError(t, err)

	select {
	case agg := <-ch:
		// 0.218 / 0.000678 = 321.53...; * 10^8 ≈ 32_153_x
		// allow some delta because the reciprocal trick rounds.
		assert.InDelta(t, int64(32_153_000_000), agg.Value, 100_000_000,
			"reciprocal mode should divide aggregate by source value")
	case <-time.After(time.Second):
		t.Fatal("expected aggregate on channel")
	}
}

// Per-feed multiplier (Definition.MultiplyBy) tests --------------------------

func newPerFeedMultiplierAggregator(t *testing.T, feeds []Feed) *LocalAggregator {
	t.Helper()
	freshness := 60_000
	return &LocalAggregator{
		Config: Config{
			ID:                100,
			Name:              "IDRX-USDT",
			FetchInterval:     2000,
			Decimals:          intPtr(8),
			FeedDataFreshness: &freshness,
		},
		Feeds:                  feeds,
		localAggregateValueMap: NewLocalAggregateValueMap(),
	}
}

func feedWithMultiplyBy(id int32, name string, multiplyBy string, reciprocal bool) Feed {
	defn := map[string]any{
		"type":                 "UniswapPool",
		"chainId":              "8453",
		"address":              "0x457C528A3d135EC387caB8848D7aFeDfAB49a82F",
		"token0Decimals":       18,
		"token1Decimals":       6,
		"multiplyBy":           multiplyBy,
		"multiplyByReciprocal": reciprocal,
	}
	raw, _ := json.Marshal(defn)
	return Feed{ID: id, Name: name, Definition: raw}
}

func feedNoMultiplier(id int32, name string) Feed {
	defn := map[string]any{
		"type":           "UniswapPool",
		"chainId":        "8217",
		"address":        "0x68dd15d01f36dc92677fcf353ff9f1474f98ce19",
		"token0Decimals": 0,
		"token1Decimals": 6,
	}
	raw, _ := json.Marshal(defn)
	return Feed{ID: id, Name: name, Definition: raw}
}

func TestApplyPerFeedMultipliers_AppliesMultiplier(t *testing.T) {
	la := newPerFeedMultiplierAggregator(t, []Feed{
		feedWithMultiplyBy(1, "PancakeSwap-IDRX-USDC", "USDC-USDT", false),
		feedNoMultiplier(2, "DragonSwap-IDRX-USDT"),
	})
	la.localAggregateValueMap.Set("USDC-USDT", 1.0001) // raw USDC-USDT

	now := time.Now()
	feeds := []*FeedData{
		{FeedID: 1, Value: 0.0000625, Timestamp: &now},
		{FeedID: 2, Value: 0.0000628, Timestamp: &now},
	}
	out := la.applyPerFeedMultipliers(feeds)

	assert.Len(t, out, 2)
	// PancakeSwap feed multiplied by 1.0001
	assert.InDelta(t, 0.0000625*1.0001, out[0].Value, 1e-12)
	// DragonSwap feed unchanged
	assert.Equal(t, 0.0000628, out[1].Value)
	// Original FeedData not mutated (latestFeedDataMap is shared).
	assert.Equal(t, 0.0000625, feeds[0].Value)
}

func TestApplyPerFeedMultipliers_Reciprocal(t *testing.T) {
	la := newPerFeedMultiplierAggregator(t, []Feed{
		feedWithMultiplyBy(1, "PancakeSwap-IDRX-USDC", "USDT-USDC", true),
	})
	la.localAggregateValueMap.Set("USDT-USDC", 0.9999)

	now := time.Now()
	feeds := []*FeedData{{FeedID: 1, Value: 0.0000625, Timestamp: &now}}
	out := la.applyPerFeedMultipliers(feeds)

	assert.Len(t, out, 1)
	assert.InDelta(t, 0.0000625/0.9999, out[0].Value, 1e-12)
}

func TestApplyPerFeedMultipliers_DropsWhenSourceMissing(t *testing.T) {
	la := newPerFeedMultiplierAggregator(t, []Feed{
		feedWithMultiplyBy(1, "PancakeSwap-IDRX-USDC", "USDC-USDT", false),
		feedNoMultiplier(2, "DragonSwap-IDRX-USDT"),
	})
	// Don't set USDC-USDT in the map.

	now := time.Now()
	feeds := []*FeedData{
		{FeedID: 1, Value: 0.0000625, Timestamp: &now},
		{FeedID: 2, Value: 0.0000628, Timestamp: &now},
	}
	out := la.applyPerFeedMultipliers(feeds)

	// Synthetic feed dropped, non-synthetic kept.
	assert.Len(t, out, 1)
	assert.Equal(t, int32(2), out[0].FeedID)
}

func TestApplyPerFeedMultipliers_DropsWhenSourceStale(t *testing.T) {
	la := newPerFeedMultiplierAggregator(t, []Feed{
		feedWithMultiplyBy(1, "PancakeSwap-IDRX-USDC", "USDC-USDT", false),
	})
	// Inject a stale entry directly so we can rewind the timestamp.
	la.localAggregateValueMap.Mu.Lock()
	la.localAggregateValueMap.Data["USDC-USDT"] = LocalAggregateValueEntry{
		Value:     1.0001,
		Timestamp: time.Now().Add(-10 * time.Minute),
	}
	la.localAggregateValueMap.Mu.Unlock()

	now := time.Now()
	feeds := []*FeedData{{FeedID: 1, Value: 0.0000625, Timestamp: &now}}
	out := la.applyPerFeedMultipliers(feeds)

	assert.Len(t, out, 0)
}

func TestApplyPerFeedMultipliers_NoMultiplyByPassesThrough(t *testing.T) {
	la := newPerFeedMultiplierAggregator(t, []Feed{
		feedNoMultiplier(1, "DragonSwap-IDRX-USDT"),
	})

	now := time.Now()
	feeds := []*FeedData{{FeedID: 1, Value: 0.0000628, Timestamp: &now}}
	out := la.applyPerFeedMultipliers(feeds)

	assert.Len(t, out, 1)
	assert.Equal(t, 0.0000628, out[0].Value)
}

func TestStreamLocalAggregate_ZeroAggregate_NoOp(t *testing.T) {
	la, ch := newLocalAggregatorForMultiply(t, nil, intPtr(60_000))

	err := la.streamLocalAggregate(context.Background(), 0)
	assert.NoError(t, err)

	select {
	case <-ch:
		t.Fatal("zero aggregate should not emit")
	case <-time.After(50 * time.Millisecond):
	}

	_, ok := la.localAggregateValueMap.Get("STG-KRW")
	assert.False(t, ok, "zero aggregate should not be cached")
}
