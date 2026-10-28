//nolint:all
package fetcher

import (
	"context"
	"errors"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/bus"
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

// single-config set with long intervals
func newTestFetcherState(app *App) *fetcherState {
	config := Config{ID: 1, Name: "TEST-USDT", FetchInterval: 3600000}
	return &fetcherState{
		fetchers:         map[int32]*Fetcher{1: NewFetcher(config, nil, app.LatestFeedDataMap)},
		localAggregators: map[int32]*LocalAggregator{1: NewLocalAggregator(config, nil, nil, app.LatestFeedDataMap, app.LocalAggregateValueMap)},
		websocketFetcher: websocketfetcher.New(),
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
	assert.Same(t, websocketFetcher, app.WebsocketFetcher)

	assert.True(t, app.Fetchers[1].isRunning)
	assert.True(t, app.LocalAggregators[1].isRunning)
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
	assert.Same(t, next.websocketFetcher, app.WebsocketFetcher)

	assert.False(t, old.fetchers[1].isRunning)
	assert.False(t, old.localAggregators[1].isRunning)

	assert.True(t, app.Fetchers[1].isRunning)
	assert.True(t, app.LocalAggregators[1].isRunning)
}

func TestLoadProxies(t *testing.T) {
	t.Setenv(EnvProxies, "")
	proxies, err := loadProxies()
	assert.NoError(t, err)
	assert.Empty(t, proxies)

	t.Setenv(EnvProxies, " http://172.0.200.20:4000 , kr=http://3.39.22.82:80")
	proxies, err = loadProxies()
	assert.NoError(t, err)
	if assert.Len(t, proxies, 2) {
		assert.Equal(t, "http://172.0.200.20:4000", proxies[0].GetProxyUrl())
		assert.Nil(t, proxies[0].Location)
		assert.Equal(t, "http://3.39.22.82:80", proxies[1].GetProxyUrl())
		if assert.NotNil(t, proxies[1].Location) {
			assert.Equal(t, "kr", *proxies[1].Location)
		}
	}

	for _, bad := range []string{"172.0.200.20:4000", "http://host", "http://host:0", "http://host:4000/path", "http://user:pw@host:4000", "kr=", "http://a:1,"} {
		t.Setenv(EnvProxies, bad)
		_, err = loadProxies()
		assert.Error(t, err, bad)
	}
}
