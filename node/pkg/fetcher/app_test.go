//nolint:all
package fetcher

import (
	"context"
	"errors"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/bus"
	"bisonai.com/miko/node/pkg/db"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/websocketfetcher"
	"github.com/stretchr/testify/assert"
)

const WAIT_SECONDS = 4 * time.Second
const DUMMY_FEED_VALUE = 1.0

func TestFetcherInitialize(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := clean(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	app := testItems.app

	err = app.initialize(ctx)
	if err != nil {
		t.Fatalf("error initializing fetcher: %v", err)
	}

	assert.Greater(t, len(app.Fetchers), 0)
	for _, adapter := range app.Fetchers {
		assert.Greater(t, len(adapter.Feeds), 0)
	}
}

func TestAppRun(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := clean(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	app := testItems.app

	err = app.initialize(ctx)
	if err != nil {
		t.Fatalf("error initializing fetcher: %v", err)
	}

	err = app.Run(ctx)
	if err != nil {
		t.Fatalf("error running fetcher: %v", err)
	}
	for _, fetcher := range app.Fetchers {
		assert.True(t, fetcher.isRunning)
	}
	for _, localAggregator := range app.LocalAggregators {
		assert.True(t, localAggregator.isRunning)
	}
	assert.True(t, app.FeedDataBulkWriter.isRunning)

	time.Sleep(WAIT_SECONDS)

	err = app.stopAll(ctx)
	if err != nil {
		t.Fatalf("error stopping fetcher: %v", err)
	}
	for _, fetcher := range app.Fetchers {
		assert.False(t, fetcher.isRunning)
	}
	for _, localAggregator := range app.LocalAggregators {
		assert.False(t, localAggregator.isRunning)
	}
	assert.False(t, app.FeedDataBulkWriter.isRunning)

	for _, fetcher := range app.Fetchers {
		for _, feed := range fetcher.Feeds {

			result, letestFeedDataErr := app.LatestFeedDataMap.GetLatestFeedData([]int32{feed.ID})
			if letestFeedDataErr != nil {
				t.Fatalf("error getting latest feed data: %v", letestFeedDataErr)
			}
			assert.NotNil(t, result)
		}
		data := <-testItems.aggChan
		assert.NotNil(t, data)
	}

	err = app.FeedDataBulkWriter.Job(ctx)
	if err != nil {
		t.Fatalf("error running FeedDataBulkWriter job: %v", err)
	}

	feedResult, err := db.QueryRows[FeedData](ctx, "SELECT * FROM feed_data", nil)
	if err != nil {
		t.Fatalf("error querying feed data: %v", err)
	}
	assert.Greater(t, len(feedResult), 0)

	localAggregateResult, err := db.QueryRows[LocalAggregate](ctx, "SELECT * FROM local_aggregates", nil)
	if err != nil {
		t.Fatalf("error querying local aggregates: %v", err)
	}
	assert.Greater(t, len(localAggregateResult), 0)
}

func TestHandleMessageRejectsOverlappingLifecycleCommand(t *testing.T) {
	app := &App{}
	// simulate a start/stop/refresh already in flight
	app.lifecycleMu.Lock()
	defer app.lifecycleMu.Unlock()

	for _, command := range []string{bus.STOP_FETCHER_APP, bus.START_FETCHER_APP, bus.REFRESH_FETCHER_APP} {
		msg := bus.Message{
			From:     bus.ADMIN,
			To:       bus.FETCHER,
			Content:  bus.MessageContent{Command: command},
			Response: make(chan bus.MessageResponse, 1),
		}
		app.handleMessage(context.Background(), msg)

		select {
		case resp := <-msg.Response:
			assert.False(t, resp.Success, command)
			assert.Equal(t, errorSentinel.ErrFetcherLifecycleInProgress.Error(), resp.Args["error"], command)
		default:
			t.Fatalf("%s: no response while lifecycle in progress", command)
		}
	}
}

// reply is delivered on the caller's (unbuffered) channel only after
// lifecycleMu is released
func TestHandleMessageRepliesAfterUnlock(t *testing.T) {
	app := New(nil)
	app.LocalAggregateBulkWriter = NewLocalAggregateBulkWriter(DefaultLocalAggregateInterval)
	app.FeedDataBulkWriter = NewFeedDataBulkWriter(DefaultFeedDataDumpInterval, app.FeedDataDumpChannel)

	msg := bus.Message{
		From:     bus.ADMIN,
		To:       bus.FETCHER,
		Content:  bus.MessageContent{Command: bus.STOP_FETCHER_APP},
		Response: make(chan bus.MessageResponse),
	}
	go app.handleMessage(context.Background(), msg)

	select {
	case resp := <-msg.Response:
		assert.True(t, resp.Success)
	case <-time.After(time.Second):
		t.Fatal("no response")
	}
	assert.True(t, app.lifecycleMu.TryLock(), "lifecycleMu still held after reply")
	app.lifecycleMu.Unlock()
}

// single-config set with long intervals, buildable without DB
func newTestFetcherState(app *App) *fetcherState {
	config := Config{ID: 1, Name: "TEST-USDT", FetchInterval: 3600000}
	localAggregateBulkWriter := NewLocalAggregateBulkWriter(time.Hour)
	localAggregateBulkWriter.localAggregatesChannel = make(chan *LocalAggregate, 1)
	return &fetcherState{
		fetchers:                 map[int32]*Fetcher{1: NewFetcher(config, nil, app.LatestFeedDataMap, app.FeedDataDumpChannel)},
		localAggregators:         map[int32]*LocalAggregator{1: NewLocalAggregator(config, nil, localAggregateBulkWriter.localAggregatesChannel, nil, app.LatestFeedDataMap, app.LocalAggregateValueMap)},
		localAggregateBulkWriter: localAggregateBulkWriter,
		feedDataBulkWriter:       NewFeedDataBulkWriter(time.Hour, app.FeedDataDumpChannel),
		websocketFetcher:         websocketfetcher.New(),
	}
}

// a refresh whose rebuild fails mid-way must leave the previous set assigned
// and running, not stopped with partial maps
func TestRefreshKeepsPreviousSetOnInitializeFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := New(nil)
	app.applyState(newTestFetcherState(app))
	if err := app.startAll(ctx); err != nil {
		t.Fatalf("error starting fetchers: %v", err)
	}
	defer app.stopAll(ctx)

	fetchers, localAggregators := app.Fetchers, app.LocalAggregators
	feedDataBulkWriter, localAggregateBulkWriter := app.FeedDataBulkWriter, app.LocalAggregateBulkWriter
	websocketFetcher := app.WebsocketFetcher

	injected := errors.New("injected build failure")
	orig := buildFetcherState
	buildFetcherState = func(*App, context.Context) (*fetcherState, error) { return nil, injected }
	defer func() { buildFetcherState = orig }()

	msg := bus.Message{
		From:     bus.ADMIN,
		To:       bus.FETCHER,
		Content:  bus.MessageContent{Command: bus.REFRESH_FETCHER_APP},
		Response: make(chan bus.MessageResponse, 1),
	}
	app.handleMessage(ctx, msg)

	select {
	case resp := <-msg.Response:
		assert.False(t, resp.Success)
		assert.Equal(t, injected.Error(), resp.Args["error"])
	default:
		t.Fatal("no response")
	}

	assert.Len(t, app.Fetchers, len(fetchers))
	assert.Same(t, fetchers[1], app.Fetchers[1])
	assert.Len(t, app.LocalAggregators, len(localAggregators))
	assert.Same(t, localAggregators[1], app.LocalAggregators[1])
	assert.Same(t, feedDataBulkWriter, app.FeedDataBulkWriter)
	assert.Same(t, localAggregateBulkWriter, app.LocalAggregateBulkWriter)
	assert.Same(t, websocketFetcher, app.WebsocketFetcher)

	assert.True(t, app.Fetchers[1].isRunning)
	assert.True(t, app.LocalAggregators[1].isRunning)
	assert.True(t, app.FeedDataBulkWriter.isRunning)
	assert.True(t, app.LocalAggregateBulkWriter.isRunning)
}

// a successful refresh stops the previous set and swaps in the new one
func TestRefreshSwapsInNewSet(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := New(nil)
	old := newTestFetcherState(app)
	app.applyState(old)
	if err := app.startAll(ctx); err != nil {
		t.Fatalf("error starting fetchers: %v", err)
	}

	next := newTestFetcherState(app)
	orig := buildFetcherState
	buildFetcherState = func(*App, context.Context) (*fetcherState, error) { return next, nil }
	defer func() { buildFetcherState = orig }()

	msg := bus.Message{
		From:     bus.ADMIN,
		To:       bus.FETCHER,
		Content:  bus.MessageContent{Command: bus.REFRESH_FETCHER_APP},
		Response: make(chan bus.MessageResponse, 1),
	}
	app.handleMessage(ctx, msg)
	defer app.stopAll(ctx)

	select {
	case resp := <-msg.Response:
		assert.True(t, resp.Success)
	default:
		t.Fatal("no response")
	}

	assert.Same(t, next.fetchers[1], app.Fetchers[1])
	assert.Same(t, next.localAggregators[1], app.LocalAggregators[1])
	assert.Same(t, next.feedDataBulkWriter, app.FeedDataBulkWriter)
	assert.Same(t, next.localAggregateBulkWriter, app.LocalAggregateBulkWriter)
	assert.Same(t, next.websocketFetcher, app.WebsocketFetcher)

	assert.False(t, old.fetchers[1].isRunning)
	assert.False(t, old.localAggregators[1].isRunning)
	assert.False(t, old.feedDataBulkWriter.isRunning)
	assert.False(t, old.localAggregateBulkWriter.isRunning)

	assert.True(t, app.Fetchers[1].isRunning)
	assert.True(t, app.LocalAggregators[1].isRunning)
	assert.True(t, app.FeedDataBulkWriter.isRunning)
	assert.True(t, app.LocalAggregateBulkWriter.isRunning)
}
