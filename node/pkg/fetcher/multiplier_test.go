//nolint:all
package fetcher

import (
	"context"
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
	t.Helper()
	ch := make(chan *LocalAggregate, 1)
	mb := bus.New(10)
	// streamLocalAggregate publishes to the AGGREGATOR channel and
	// returns ErrBusChannelNotFound otherwise; subscribe a no-op consumer
	// to keep the publish path happy in tests.
	mb.Subscribe(bus.AGGREGATOR)
	return &LocalAggregator{
		Config: Config{
			ID:                42,
			Name:              "STG-KRW",
			FetchInterval:     2000,
			Decimals:          intPtr(8),
			FeedDataFreshness: freshnessMs,
			MultiplyBy:        multiplyBy,
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
