//nolint:all
package fetcher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/bus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFetcherStopStartWhileJobInFlight checks that a STOP->START landing while
// the old loop is mid-job leaves exactly one fetch loop running (#2793).
func TestFetcherStopStartWhileJobInFlight(t *testing.T) {
	const interval = 20 * time.Millisecond

	var requests atomic.Int64
	firstArrived := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		blockedHere := false
		once.Do(func() {
			blockedHere = true
			close(firstArrived)
		})
		if blockedHere {
			// Keep the first job in flight until the test releases it.
			<-release
		}
		w.Write([]byte(`{"price": "1.5"}`))
	}))
	defer srv.Close()
	// Unblock the handler before srv.Close (defers run LIFO) so a failed
	// require can't hang Close on the in-flight request.
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unblock()

	definition, err := json.Marshal(map[string]any{
		"url":      srv.URL,
		"headers":  map[string]string{"Content-Type": "application/json"},
		"method":   "GET",
		"reducers": []map[string]any{{"function": "PARSE", "args": []string{"price"}}},
	})
	require.NoError(t, err)

	latest := &LatestFeedDataMap{FeedDataMap: make(map[int32]*FeedData)}
	f := NewFetcher(
		Config{ID: 1, Name: "TEST-USDT", FetchInterval: int32(interval / time.Millisecond)},
		[]Feed{{ID: 1, Name: "test-feed", Definition: definition, ConfigID: 1}},
		latest,
	)
	app := &App{}
	// Parent ctx cancelled at test end so loops never outlive a failed test.
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	require.NoError(t, app.startFetcher(ctx, f))

	select {
	case <-firstArrived:
	case <-time.After(5 * time.Second):
		t.Fatal("first fetch never reached the server")
	}

	// STOP -> START while the first job is still blocked in the HTTP call.
	require.NoError(t, app.stopFetcher(ctx, f))
	require.NoError(t, app.startFetcher(ctx, f))
	unblock()

	// Let the old goroutine finish its job and observe its cancelled ctx.
	time.Sleep(5 * interval)

	const window = 30 * interval
	before := requests.Load()
	time.Sleep(window)
	got := requests.Load() - before

	require.NoError(t, app.stopFetcher(ctx, f))

	// One loop fires at most once per tick (the ticker buffers a single tick,
	// so delays can't burst); two loops would roughly double that. Allow 1.5x
	// headroom for CI jitter while still catching a second loop.
	ceiling := int64(window/interval) * 3 / 2
	assert.Greater(t, got, int64(0), "new loop should be fetching")
	assert.LessOrEqual(t, got, ceiling, "more requests than one loop can make: old loop still running")
}

// TestLocalAggregatorStopStartLeavesOneLoop checks that rapid STOP->START
// cycles leave exactly one local aggregator loop running (#2793).
func TestLocalAggregatorStopStartLeavesOneLoop(t *testing.T) {
	const interval = 20 * time.Millisecond
	t.Setenv("LOCAL_AGGREGATE_INTERVAL", interval.String())

	mb := bus.New(1000)
	msgs := mb.Subscribe(bus.AGGREGATOR)
	var published atomic.Int64
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-msgs:
				published.Add(1)
			}
		}
	}()

	now := time.Now()
	latest := &LatestFeedDataMap{FeedDataMap: map[int32]*FeedData{
		1: {FeedID: 1, Value: 1.5, Timestamp: &now},
	}}
	c := NewLocalAggregator(
		Config{ID: 1, Name: "TEST-USDT"},
		[]Feed{{ID: 1, Name: "test-feed", ConfigID: 1}},
		mb,
		latest,
		nil,
	)
	app := &App{}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	require.NoError(t, app.startLocalAggregator(ctx, c))
	for i := 0; i < 50; i++ {
		require.NoError(t, app.stopLocalAggregator(ctx, c))
		require.NoError(t, app.startLocalAggregator(ctx, c))
	}

	// Let Job goroutines spawned by earlier loops drain before measuring.
	time.Sleep(15 * interval)

	const window = 30 * interval
	before := published.Load()
	time.Sleep(window)
	got := published.Load() - before

	require.NoError(t, app.stopLocalAggregator(ctx, c))

	ceiling := int64(window/interval) * 3 / 2
	assert.Greater(t, got, int64(0), "new loop should be aggregating")
	assert.LessOrEqual(t, got, ceiling, "more aggregates than one loop can emit: old loop still running")
}
