package websocketfetcher

import (
	"bytes"
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
	"bisonai.com/miko/node/pkg/chain/websocketchainreader"
	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubFetcher struct{ _ int }

func (s *stubFetcher) Run(context.Context) {}

type stubClient struct {
	utils.ClientInterface
	closed bool
}

func (s *stubClient) Close() { s.closed = true }

// Init is re-run on refresh; fetchers from a previous Init must not survive.
func TestInitResetsFetchersOnReinit(t *testing.T) {
	// Force initializeDex to fail fast (no dial, no DB) after initializeCex.
	for _, k := range []string{"KAIA_WEBSOCKET_URL", "ETH_WEBSOCKET_URL", "BSC_WEBSOCKET_URL", "POLYGON_WEBSOCKET_URL"} {
		t.Setenv(k, "")
	}

	feeds := []common.Feed{
		{ID: 1, Name: "binance-wss-BTC-USDT", Definition: json.RawMessage(`{"type": "wss", "provider": "binance", "base": "btc", "quote": "usdt"}`), ConfigID: 1},
		{ID: 2, Name: "coinbase-wss-ADA-USDT", Definition: json.RawMessage(`{"type": "wss", "provider": "coinbase", "base": "ada", "quote": "usdt"}`), ConfigID: 2},
	}
	factory := func(context.Context, ...common.FetcherOption) (common.FetcherInterface, error) {
		return &stubFetcher{}, nil
	}
	cexFactories := map[string]func(context.Context, ...common.FetcherOption) (common.FetcherInterface, error){
		"binance":  factory,
		"coinbase": factory,
	}
	opts := []AppOption{WithSetFromConfig(false), WithFeeds(feeds), WithCexFactories(cexFactories)}

	ctx := context.Background()
	a := New()

	_ = a.Init(ctx, opts...) // dex init errors on unset URLs; cex fetchers already built
	require.Len(t, a.fetchers, 2)
	first := append([]common.FetcherInterface(nil), a.fetchers...)

	old := &stubClient{}
	a.chainReader = &websocketchainreader.ChainReader{KaiaClient: old}

	_ = a.Init(ctx, opts...)
	require.Len(t, a.fetchers, 2)
	for _, f := range a.fetchers {
		for _, o := range first {
			assert.NotSame(t, o, f, "stale fetcher kept after re-Init")
		}
	}
	assert.True(t, old.closed, "old chain reader not closed on re-Init")
	assert.Nil(t, a.chainReader)
}

// A feed whose provider has no registered factory (e.g. a removed exchange
// still in config) must be skipped without failing init of other providers.
func TestInitializeCexSkipsUnknownProvider(t *testing.T) {
	var created []string
	cexFactories := map[string]func(context.Context, ...common.FetcherOption) (common.FetcherInterface, error){
		"binance": func(context.Context, ...common.FetcherOption) (common.FetcherInterface, error) {
			created = append(created, "binance")
			return &stubFetcher{}, nil
		},
	}
	feeds := []common.Feed{
		{ID: 1, Name: "binance-wss-BTC-USDT", Definition: json.RawMessage(`{"type": "wss", "provider": "binance", "base": "btc", "quote": "usdt"}`), ConfigID: 1},
		{ID: 2, Name: "coinex-wss-BTC-USDT", Definition: json.RawMessage(`{"type": "wss", "provider": "coinex", "base": "btc", "quote": "usdt"}`), ConfigID: 2},
	}

	var logBuf bytes.Buffer
	prevLogger := log.Logger
	log.Logger = zerolog.New(&logBuf)
	t.Cleanup(func() { log.Logger = prevLogger })

	a := New()
	err := a.initializeCex(context.Background(), AppConfig{
		Feeds:        feeds,
		CexFactories: cexFactories,
		BufferSize:   1,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"binance"}, created)
	assert.Len(t, a.fetchers, 1)
	assert.Contains(t, logBuf.String(), `"provider":"coinex"`, "unknown provider not logged")
}

type blockingFetcher struct {
	runs  atomic.Int32
	exits chan struct{}
}

func (f *blockingFetcher) Run(ctx context.Context) {
	f.runs.Add(1)
	<-ctx.Done()
	f.exits <- struct{}{}
}

// Start must set cancel before returning and must not orphan a running run
// when called again (fetcher START while already running).
func TestAppStartIdempotentAndStoppable(t *testing.T) {
	fetcher := &blockingFetcher{exits: make(chan struct{}, 4)}
	app := &App{storeInterval: 10 * time.Millisecond, fetchers: []common.FetcherInterface{fetcher}}

	app.Start(context.Background())
	require.NotNil(t, app.cancel)
	app.Start(context.Background())

	app.Stop()
	assert.Nil(t, app.cancel)

	// every Run started must exit after a single Stop
	select {
	case <-fetcher.exits:
	case <-time.After(time.Second):
		t.Fatal("fetcher run not stopped")
	}
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(1), fetcher.runs.Load(), "second Start launched another run")
	assert.Len(t, fetcher.exits, 0, "orphaned run")
}
