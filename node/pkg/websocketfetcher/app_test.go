package websocketfetcher

import (
	"context"
	"testing"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubFetcher struct{}

func (stubFetcher) Run(context.Context) {}

// A feed whose provider has no registered factory (e.g. a removed exchange
// still in config) must be skipped without failing init of other providers.
func TestInitializeCexSkipsUnknownProvider(t *testing.T) {
	var created []string
	factories := map[string]func(context.Context, ...common.FetcherOption) (common.FetcherInterface, error){
		"binance": func(context.Context, ...common.FetcherOption) (common.FetcherInterface, error) {
			created = append(created, "binance")
			return stubFetcher{}, nil
		},
	}
	feeds := []common.Feed{
		{ID: 1, Name: "binance-wss-BTC-USDT", Definition: []byte(`{"type":"wss","provider":"binance","base":"btc","quote":"usdt"}`)},
		{ID: 2, Name: "coinex-wss-BTC-USDT", Definition: []byte(`{"type":"wss","provider":"coinex","base":"btc","quote":"usdt"}`)},
	}

	a := New()
	err := a.initializeCex(context.Background(), AppConfig{
		SetFromDB:    false,
		Feeds:        feeds,
		CexFactories: factories,
		BufferSize:   1,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"binance"}, created)
	assert.Len(t, a.fetchers, 1)
}
