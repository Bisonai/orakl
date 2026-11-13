package collector

import (
	"context"
	"sync"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/aggregator"
	"bisonai.com/miko/node/pkg/common/keys"
	dalcommon "bisonai.com/miko/node/pkg/dal/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSubscriber struct {
	mu    sync.Mutex
	calls [][]string
	ctxs  []context.Context
	err   error
	ready chan struct{} // nil: connected at once
}

func (f *fakeSubscriber) subscribe(ctx context.Context, topics []string) (<-chan struct{}, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	f.calls = append(f.calls, topics)
	f.ctxs = append(f.ctxs, ctx)
	ready := f.ready
	if ready == nil {
		closed := make(chan struct{})
		close(closed)
		ready = closed
	}
	return ready, nil
}

// newTestCollector returns a started-looking collector (no redis, no chain).
func newTestCollector(t *testing.T, names ...string) (*Collector, *fakeSubscriber) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	f := &fakeSubscriber{}
	c := &Collector{
		OutgoingStream:   map[string]chan *dalcommon.OutgoingSubmissionData{},
		FeedHashes:       map[string][]byte{},
		LatestTimestamps: map[string]time.Time{},
		LatestData:       map[string]*dalcommon.OutgoingSubmissionData{},
		Configs:          map[string]Config{},
		runCtx:           ctx,
		subscribe:        f.subscribe,
	}
	for _, n := range names {
		c.OutgoingStream[n] = make(chan *dalcommon.OutgoingSubmissionData, OutgoingStreamSize)
		c.FeedHashes[n] = []byte(n)
		c.Configs[n] = Config{Name: n}
		c.LatestData[n] = &dalcommon.OutgoingSubmissionData{Symbol: n}
	}
	return c, f
}

func cfgs(names ...string) []Config {
	out := make([]Config, len(names))
	for i, n := range names {
		out[i] = Config{Name: n}
	}
	return out
}

func TestUpdateConfigsAddRemove(t *testing.T) {
	c, f := newTestCollector(t, "A-B", "C-D")
	oldStream := c.Streams()["C-D"]

	added, removed, err := c.UpdateConfigs(context.Background(), cfgs("A-B", "E-F"))
	require.NoError(t, err)
	assert.Contains(t, added, "E-F")
	assert.Len(t, added, 1)
	assert.Equal(t, []string{"C-D"}, removed)

	require.Len(t, f.calls, 1)
	assert.Equal(t, []string{keys.SubmissionDataStreamKey("A-B"), keys.SubmissionDataStreamKey("E-F")}, f.calls[0])
	assert.Contains(t, c.Streams(), "E-F")
	// removed symbol kept until RemoveSymbols
	assert.Contains(t, c.Streams(), "C-D")

	c.RemoveSymbols(removed)
	assert.NotContains(t, c.Streams(), "C-D")
	_, err = c.GetLatestData("C-D")
	assert.Error(t, err)
	_, ok := <-oldStream
	assert.False(t, ok, "removed stream must be closed")

	// unchanged set: no resubscribe
	added, removed, err = c.UpdateConfigs(context.Background(), cfgs("A-B", "E-F"))
	require.NoError(t, err)
	assert.Empty(t, added)
	assert.Empty(t, removed)
	assert.Len(t, f.calls, 1)
}

func TestUpdateConfigsRefreshesKeptConfig(t *testing.T) {
	c, f := newTestCollector(t, "A-B")
	decimals := 4
	_, _, err := c.UpdateConfigs(context.Background(), []Config{{Name: "A-B", Decimals: &decimals}})
	require.NoError(t, err)
	assert.Equal(t, 4, *c.Configs["A-B"].Decimals)
	assert.Empty(t, f.calls)
}

func TestUpdateConfigsSubscribeErrorChangesNothing(t *testing.T) {
	c, f := newTestCollector(t, "A-B")
	f.err = assert.AnError
	_, _, err := c.UpdateConfigs(context.Background(), cfgs("A-B", "E-F"))
	assert.ErrorIs(t, err, assert.AnError)
	assert.NotContains(t, c.Streams(), "E-F", "added symbol rolled back")

	// retried later: still detected as an addition
	f.err = nil
	added, _, err := c.UpdateConfigs(context.Background(), cfgs("A-B", "E-F"))
	require.NoError(t, err)
	assert.Contains(t, added, "E-F")
}

func TestUpdateConfigsEmptyRefused(t *testing.T) {
	c, f := newTestCollector(t, "A-B")
	_, _, err := c.UpdateConfigs(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, c.Streams(), "A-B")
	assert.Empty(t, f.calls)
}

func TestResubscribeCancelsPreviousAfterGrace(t *testing.T) {
	orig := ResubscribeGrace
	ResubscribeGrace = 200 * time.Millisecond
	t.Cleanup(func() { ResubscribeGrace = orig })
	c, f := newTestCollector(t, "A-B")
	_, _, err := c.UpdateConfigs(context.Background(), cfgs("A-B", "C-D"))
	require.NoError(t, err)
	_, _, err = c.UpdateConfigs(context.Background(), cfgs("A-B", "C-D", "E-F"))
	require.NoError(t, err)
	require.Len(t, f.ctxs, 2)
	assert.NoError(t, f.ctxs[1].Err(), "live subscription must stay open")
	require.Eventually(t, func() bool { return f.ctxs[0].Err() != nil }, ResubscribeGrace+2*time.Second, 50*time.Millisecond)
}

func TestRemovedSymbolDataIgnored(t *testing.T) {
	c, _ := newTestCollector(t, "A-B")
	c.RemoveSymbols([]string{"A-B"})
	data := &aggregator.SubmissionData{Symbol: "A-B", GlobalAggregate: aggregator.GlobalAggregate{Timestamp: time.Now()}}
	assert.False(t, c.isNewer(data))
	c.mu.RLock()
	defer c.mu.RUnlock()
	assert.NotContains(t, c.LatestTimestamps, "A-B")
}

func TestResubscribeKeepsPreviousUntilConnected(t *testing.T) {
	orig := ResubscribeGrace
	ResubscribeGrace = 50 * time.Millisecond
	t.Cleanup(func() { ResubscribeGrace = orig })

	c, f := newTestCollector(t, "A-B")
	_, _, err := c.UpdateConfigs(context.Background(), cfgs("A-B", "C-D"))
	require.NoError(t, err)

	f.ready = make(chan struct{}) // redis unreachable for the next subscription
	_, _, err = c.UpdateConfigs(context.Background(), cfgs("A-B", "C-D", "E-F"))
	require.NoError(t, err)
	time.Sleep(300 * time.Millisecond)
	assert.NoError(t, f.ctxs[0].Err(), "previous subscription must stay while the new one is not connected")

	close(f.ready)
	require.Eventually(t, func() bool { return f.ctxs[0].Err() != nil }, 2*time.Second, 20*time.Millisecond)
}

func TestUpdateConfigsSubscribeErrorKeepsKeptConfig(t *testing.T) {
	c, f := newTestCollector(t, "A-B")
	f.err = assert.AnError
	decimals := 4
	_, _, err := c.UpdateConfigs(context.Background(), []Config{{Name: "A-B", Decimals: &decimals}, {Name: "E-F"}})
	assert.Error(t, err)
	assert.Nil(t, c.Configs["A-B"].Decimals)
}
