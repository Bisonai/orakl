//nolint:all
package fetcher

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"net/http"

	"bisonai.com/orakl/node/pkg/admin/tests"
	"bisonai.com/orakl/node/pkg/db"
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

func TestFetcherFetchAndInsertAdapter(t *testing.T) {
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

	app.initialize(ctx)

	for _, fetcher := range app.Fetchers {
		fetcher.fetchAndInsert(ctx, app.ChainHelpers, app.Proxies)
	}
	if err != nil {
		t.Fatalf("error running adapter: %v", err)
	}

	for _, fetcher := range app.Fetchers {
		pgResult, err := db.QueryRow[Aggregate](ctx, "SELECT * FROM local_aggregates WHERE name = @name", map[string]any{"name": fetcher.Name})
		if err != nil {
			t.Fatalf("error reading from db: %v", err)
		}
		assert.NotNil(t, pgResult)

		feedPgResult, err := db.QueryRows[FeedDataFromDB](ctx, "SELECT * FROM feed_data WHERE adapter_id = @adapter_id", map[string]any{"adapter_id": fetcher.Adapter.ID})
		if err != nil {
			t.Fatalf("error reading from db: %v", err)
		}
		assert.Greater(t, len(feedPgResult), 0)

		rdbResult, err := db.Get(ctx, "localAggregate:"+fetcher.Name)
		if err != nil {
			t.Fatalf("error reading from redis: %v", err)
		}
		assert.NotNil(t, rdbResult)
		var redisAgg redisAggregate
		err = json.Unmarshal([]byte(rdbResult), &redisAgg)
		if err != nil {
			t.Fatalf("error unmarshalling from redis: %v", err)
		}
		assert.NotNil(t, redisAgg)
		assert.NotNil(t, redisAgg.Value)

		err = db.Del(ctx, "localAggregate:"+fetcher.Name)
		if err != nil {
			t.Fatalf("error removing from redis: %v", err)
		}
	}
}
