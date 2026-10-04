package websocketfetcher

import (
	"context"
	"encoding/json"
	"testing"

	"bisonai.com/miko/node/pkg/chain/utils"
	"bisonai.com/miko/node/pkg/chain/websocketchainreader"
	"bisonai.com/miko/node/pkg/websocketfetcher/common"
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
	opts := []AppOption{WithSetFromDB(false), WithFeeds(feeds), WithCexFactories(cexFactories)}

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
