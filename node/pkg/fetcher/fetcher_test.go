//nolint:all
package fetcher

import (
	"context"
	"encoding/json"
	"testing"

	"net/http"

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

	for _, fetcher := range app.Fetchers {
		for _, feed := range fetcher.Feeds {
			res, latestFeedDataErr := app.LatestFeedDataMap.GetLatestFeedData([]int32{feed.ID})
			if latestFeedDataErr != nil {
				t.Fatalf("error fetching feed data: %v", latestFeedDataErr)
			}
			assert.NotNil(t, res)
		}
	}
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
	t.Setenv(EnvProxies, "http://localhost:8088")
	defer func() {
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

func TestFetcherFilterProxyByLocation(t *testing.T) {
	uk := "uk"
	us := "us"
	kr := "kr"
	proxies := []Proxy{
		{ID: 1, Protocol: "http", Host: "localhost", Port: 8080, Location: &uk},
		{ID: 2, Protocol: "http", Host: "localhost", Port: 8081, Location: &us},
		{ID: 3, Protocol: "http", Host: "localhost", Port: 8082, Location: &kr},
	}

	testMap := &LatestFeedDataMap{
		FeedDataMap: make(map[int32]*FeedData),
	}

	fetcher := NewFetcher(Config{}, []Feed{}, testMap)

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
