//nolint:all
package fetcher

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"net/http"

	"bisonai.com/orakl/node/pkg/admin/tests"
	"github.com/elazarl/goproxy"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestFetcherFetch(t *testing.T) {
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

	for _, fetcher := range app.Fetchers {
		result, err := fetcher.fetch(app.ChainHelpers, app.Proxies)
		if err != nil {
			t.Fatalf("error fetching: %v", err)
		}
		assert.Greater(t, len(result), 0)
	}
}

func TestFetcherFetchProxy(t *testing.T) {
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

	proxyServer := goproxy.NewProxyHttpServer()
	srv := &http.Server{
		Addr:    ":8088",
		Handler: proxyServer,
	}
	go func() {
		if proxyServeErr := srv.ListenAndServe(); proxyServeErr != http.ErrServerClosed {
			// Unexpected server shutdown
			log.Fatal().Err(proxyServeErr).Msg("unexpected server shutdown")
		}
	}()

	proxy, err := tests.PostRequest[Proxy](testItems.admin, "/api/v1/proxy", map[string]any{"protocol": "http", "host": "localhost", "port": 8088})
	if err != nil {
		t.Fatalf("error creating proxy: %v", err)
	}

	err = app.initialize(ctx)
	if err != nil {
		t.Fatalf("error initializing fetcher: %v", err)
	}
	log.Info().Msg("initialized")
	for _, fetcher := range app.Fetchers {
		result, fetchErr := fetcher.fetch(app.ChainHelpers, app.Proxies)
		if fetchErr != nil {
			t.Fatalf("error fetching: %v", fetchErr)
		}
		assert.Greater(t, len(result), 0)
	}

	_, err = tests.DeleteRequest[Proxy](testItems.admin, "/api/v1/proxy/"+strconv.FormatInt(proxy.ID, 10), nil)
	if err != nil {
		t.Fatalf("error cleaning up proxy: %v", err)
	}

	if err = srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("unexpected server shutdown")
	}

}

func TestFetchSingle(t *testing.T) {
	t.Skip() // test fails if data provider refuses connection
	ctx := context.Background()
	rawDefinition := `
	{
        "url": "https://api.bybit.com/derivatives/v3/public/tickers?symbol=ADAUSDT",
        "headers": {
          "Content-Type": "application/json"
        },
        "method": "GET",
        "reducers": [
          {
            "function": "PARSE",
            "args": [
              "result",
              "list"
            ]
          },
          {
            "function": "INDEX",
            "args": 0
          },
          {
            "function": "PARSE",
            "args": [
              "lastPrice"
            ]
          },
          {
            "function": "POW10",
            "args": 8
          },
          {
            "function": "ROUND"
          }
        ]
	}`
	definition := new(Definition)
	err := json.Unmarshal([]byte(rawDefinition), &definition)
	if err != nil {
		t.Fatalf("error unmarshalling definition: %v", err)
	}

	result, err := FetchSingle(ctx, definition)
	if err != nil {
		t.Fatalf("error fetching single: %v", err)
	}
	assert.Greater(t, result, float64(0))
}
