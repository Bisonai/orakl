//nolint:all
package fetcher

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"net/http"

	"bisonai.com/orakl/node/pkg/admin/tests"
	"bisonai.com/orakl/node/pkg/common/keys"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/elazarl/goproxy"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestFetcherRun(t *testing.T) {
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
		fetcher.Run(ctx, app.Proxies)
	}

	for _, fetcher := range app.Fetchers {
		assert.True(t, fetcher.isRunning)
	}

	for _, fetcher := range app.Fetchers {
		fetcher.cancel()
	}

	defer func() {
		db.Del(ctx, keys.FeedDataBufferKey())
		for _, fetcher := range app.Fetchers {
			for _, feed := range fetcher.Feeds {
				db.Del(ctx, keys.LatestFeedDataKey(feed.ID))
			}
		}
	}()
}

func TestFetcherFetcherJob(t *testing.T) {
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
		jobErr := fetcher.fetcherJob(ctx, app.Proxies)
		if jobErr != nil {
			t.Fatalf("error fetching: %v", jobErr)
		}
	}
	defer db.Del(ctx, keys.FeedDataBufferKey())

	for _, fetcher := range app.Fetchers {
		for _, feed := range fetcher.Feeds {
			res, latestFeedDataErr := db.GetObject[*FeedData](ctx, keys.LatestFeedDataKey(feed.ID))
			if latestFeedDataErr != nil {
				t.Fatalf("error fetching feed data: %v", latestFeedDataErr)
			}
			assert.NotNil(t, res)
			defer db.Del(ctx, keys.LatestFeedDataKey(feed.ID))
		}
	}

	buffer, err := db.LRangeObject[FeedData](ctx, keys.FeedDataBufferKey(), 0, -1)
	if err != nil {
		t.Fatalf("error fetching buffer: %v", err)
	}
	assert.Greater(t, len(buffer), 0)
}

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
		result, err := fetcher.fetch(app.Proxies)
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
	defer func() {
		_, err = tests.DeleteRequest[Proxy](testItems.admin, "/api/v1/proxy/"+strconv.FormatInt(proxy.ID, 10), nil)
		if err != nil {
			t.Fatalf("error cleaning up proxy: %v", err)
		}

		if err = srv.Shutdown(ctx); err != nil {
			log.Fatal().Err(err).Msg("unexpected server shutdown")
		}
	}()

	err = app.initialize(ctx)
	if err != nil {
		t.Fatalf("error initializing fetcher: %v", err)
	}
	log.Info().Msg("initialized")
	for _, fetcher := range app.Fetchers {
		result, fetchErr := fetcher.fetch(app.Proxies)
		if fetchErr != nil {
			t.Fatalf("error fetching: %v", fetchErr)
		}
		assert.Greater(t, len(result), 0)
	}

}

func TestFetcherCex(t *testing.T) {
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
		for _, feed := range fetcher.Feeds {
			definition := new(Definition)

			err := json.Unmarshal(feed.Definition, &definition)
			if err != nil {
				t.Fatalf("error unmarshalling definition: %v", err)
			}
			if definition.Type != nil {
				continue
			}

			result, err := fetcher.cex(definition, app.Proxies)
			if err != nil {
				t.Fatalf("error fetching: %v", err)
			}
			assert.Greater(t, result, float64(0))
		}
	}
}

func TestRequestFeed(t *testing.T) {
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
		for _, feed := range fetcher.Feeds {
			definition := new(Definition)
			err := json.Unmarshal(feed.Definition, &definition)
			if err != nil {
				t.Fatalf("error unmarshalling definition: %v", err)
			}
			if definition.Type != nil {
				continue
			}

			result, err := fetcher.requestFeed(definition, app.Proxies)
			if err != nil {
				t.Fatalf("error fetching: %v", err)
			}
			assert.NotEqual(t, result, nil)
		}
	}
}

func TestFetcherRequestWithoutProxy(t *testing.T) {
	// Being tested in TestFetcherFetch
	t.Skip()
}

func TestFetcherRequestWithProxy(t *testing.T) {
	// Being tested in TestFetcherFetchProxy
	t.Skip()
}

func TestFetcherFilterProxyByLocation(t *testing.T) {
	uk := "uk"
	us := "us"
	kr := "kr"
	proxies := []Proxy{
		{ID: 1, Protocol: "http", Host: "localhost", Port: 8080, Location: &uk},
		{ID: 2, Protocol: "http", Host: "localhost", Port: 8081, Location: &us},
		{ID: 3, Protocol: "http", Host: "localhost", Port: 8082, Location: &kr},
	}

	fetcher := NewFetcher(Config{}, []Feed{})

	res := fetcher.filterProxyByLocation(proxies, uk)
	assert.Greater(t, len(res), 0)
	assert.Equal(t, res[0], proxies[0])

	res = fetcher.filterProxyByLocation(proxies, us)
	assert.Greater(t, len(res), 0)
	assert.Equal(t, res[0], proxies[1])

	res = fetcher.filterProxyByLocation(proxies, kr)
	assert.Greater(t, len(res), 0)
	assert.Equal(t, res[0], proxies[2])
}
