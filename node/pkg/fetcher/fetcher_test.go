//nolint:all
package fetcher

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/admin/tests"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/stretchr/testify/assert"
)

const WAIT_SECONDS = 4 * time.Second

func TestFetcherInitialize(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer clean()

	app := testItems.fetcher

	err = app.initialize(ctx)
	if err != nil {
		t.Fatalf("error initializing fetcher: %v", err)
	}

	assert.Greater(t, len(app.Fetchers), 0)
	for _, adapter := range app.Fetchers {
		assert.Greater(t, len(adapter.Feeds), 0)
	}

}

func TestFetcherRun(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer clean()

	app := testItems.fetcher

	err = app.initialize(ctx)
	if err != nil {
		t.Fatalf("error initializing fetcher: %v", err)
	}

	rowsBefore, err := db.QueryRows[Aggregate](ctx, "SELECT * FROM local_aggregates", nil)
	if err != nil {
		t.Fatalf("error reading from db: %v", err)
	}
	assert.Equal(t, 0, len(rowsBefore))

	err = app.Run(ctx)
	if err != nil {
		t.Fatalf("error running fetcher: %v", err)
	}

	for _, fetcher := range app.Fetchers {
		assert.True(t, fetcher.isRunning)
	}

	// wait for fetcher to run
	time.Sleep(WAIT_SECONDS)

	// stop running after 2 seconds
	for _, fetcher := range app.Fetchers {
		app.stopFetcher(ctx, fetcher)
		assert.False(t, fetcher.isRunning)
	}

	rowsAfter, err := db.QueryRows[Aggregate](ctx, "SELECT * FROM local_aggregates", nil)
	if err != nil {
		t.Fatalf("error reading from db: %v", err)
	}
	assert.Greater(t, len(rowsAfter), len(rowsBefore))

	// clean up db
	err = db.QueryWithoutResult(ctx, "DELETE FROM local_aggregates", nil)
	if err != nil {
		t.Fatalf("error cleaning up from db: %v", err)
	}

	for _, fetcher := range app.Fetchers {
		rdbResult, err := db.Get(ctx, "latestAggregate:"+fetcher.Name)
		if err != nil {
			t.Fatalf("error reading from redis: %v", err)
		}
		assert.NotNil(t, rdbResult)

		err = db.Del(ctx, "latestAggregate:"+fetcher.Name)
		if err != nil {
			t.Fatalf("error removing from redis: %v", err)
		}
	}
}

func TestFetcherFetcherStart(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer clean()

	app := testItems.fetcher

	err = app.initialize(ctx)
	if err != nil {
		t.Fatalf("error initializing fetcher: %v", err)
	}

	rowsBefore, err := db.QueryRows[Aggregate](ctx, "SELECT * FROM local_aggregates", nil)
	if err != nil {
		t.Fatalf("error reading from db: %v", err)
	}
	assert.Equal(t, 0, len(rowsBefore))

	for _, fetcher := range app.Fetchers {
		err = app.startFetcher(ctx, fetcher)
		if err != nil {
			t.Fatalf("error starting adapter: %v", err)
		}
		assert.True(t, fetcher.isRunning)
	}

	// wait for fetcher to run
	time.Sleep(WAIT_SECONDS)

	// stop running after 2 seconds
	for _, fetcher := range app.Fetchers {
		app.stopFetcher(ctx, fetcher)
		assert.False(t, fetcher.isRunning)
	}

	rowsAfter, err := db.QueryRows[Aggregate](ctx, "SELECT * FROM local_aggregates", nil)
	if err != nil {
		t.Fatalf("error reading from db: %v", err)
	}
	assert.Greater(t, len(rowsAfter), len(rowsBefore))

	// clean up db
	err = db.QueryWithoutResult(ctx, "DELETE FROM local_aggregates", nil)
	if err != nil {
		t.Fatalf("error cleaning up from db: %v", err)
	}

	// check rdb and cleanup rdb
	for _, fetcher := range app.Fetchers {
		rdbResult, err := db.Get(ctx, "latestAggregate:"+fetcher.Name)
		if err != nil {
			t.Fatalf("error reading from redis: %v", err)
		}
		assert.NotNil(t, rdbResult)

		err = db.Del(ctx, "latestAggregate:"+fetcher.Name)
		if err != nil {
			t.Fatalf("error removing from redis: %v", err)
		}
	}
}

func TestFetcherFetcherStop(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer clean()

	app := testItems.fetcher

	err = app.initialize(ctx)
	if err != nil {
		t.Fatalf("error initializing fetcher: %v", err)
	}

	// first start adapters to stop
	for _, fetcher := range app.Fetchers {
		err = app.startFetcher(ctx, fetcher)
		if err != nil {
			t.Fatalf("error starting adapter: %v", err)
		}
		assert.True(t, fetcher.isRunning)
	}

	// wait for fetcher to run
	time.Sleep(WAIT_SECONDS)

	// stop adapters
	for _, fetcher := range app.Fetchers {
		app.stopFetcher(ctx, fetcher)
		assert.False(t, fetcher.isRunning)
	}

	time.Sleep(WAIT_SECONDS / 2)
	rowsBefore, err := db.QueryRows[Aggregate](ctx, "SELECT * FROM local_aggregates", nil)
	if err != nil {
		t.Fatalf("error reading from db: %v", err)
	}
	assert.Greater(t, len(rowsBefore), 0)
	time.Sleep(WAIT_SECONDS / 2)

	// no rows should be added after stopping
	rowsAfter, err := db.QueryRows[Aggregate](ctx, "SELECT * FROM local_aggregates", nil)
	if err != nil {
		t.Fatalf("error reading from db: %v", err)
	}
	assert.Equal(t, len(rowsAfter), len(rowsBefore))

	// clean up db
	err = db.QueryWithoutResult(ctx, "DELETE FROM local_aggregates", nil)
	if err != nil {
		t.Fatalf("error cleaning up from db: %v", err)
	}

	// check rdb and cleanup rdb
	for _, fetcher := range app.Fetchers {
		rdbResult, err := db.Get(ctx, "latestAggregate:"+fetcher.Name)
		if err != nil {
			t.Fatalf("error reading from redis: %v", err)
		}
		assert.NotNil(t, rdbResult)

		err = db.Del(ctx, "latestAggregate:"+fetcher.Name)
		if err != nil {
			t.Fatalf("error removing from redis: %v", err)
		}
	}
}

func TestFetcherFetcherStartById(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer clean()

	app := testItems.fetcher

	err = app.initialize(ctx)
	if err != nil {
		t.Fatalf("error initializing fetcher: %v", err)
	}

	app.subscribe(ctx)

	rowsBefore, err := db.QueryRows[Aggregate](ctx, "SELECT * FROM local_aggregates", nil)
	if err != nil {
		t.Fatalf("error reading from db: %v", err)
	}
	assert.Equal(t, 0, len(rowsBefore))

	for _, fetcher := range app.Fetchers {
		result, _err := tests.PostRequest[Adapter](testItems.app, "/api/v1/adapter/activate/"+strconv.FormatInt(fetcher.ID, 10), nil)
		if _err != nil {
			t.Fatalf("error starting adapter: %v", _err)
		}
		assert.True(t, result.Active)
	}

	for _, fetcher := range app.Fetchers {
		assert.True(t, fetcher.isRunning)
	}

}

func TestFetcherFetcherStopById(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer clean()

	app := testItems.fetcher

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

	for _, fetcher := range app.Fetchers {
		result, _err := tests.PostRequest[Adapter](testItems.app, "/api/v1/adapter/deactivate/"+strconv.FormatInt(fetcher.ID, 10), nil)
		if _err != nil {
			t.Fatalf("error stopping adapter: %v", _err)
		}
		assert.False(t, result.Active)

	}

	for _, fetcher := range app.Fetchers {
		assert.False(t, fetcher.isRunning)
	}
}

func TestFetcherFetch(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer clean()

	app := testItems.fetcher

	err = app.initialize(ctx)
	if err != nil {
		t.Fatalf("error initializing fetcher: %v", err)
	}

	for _, fetcher := range app.Fetchers {
		result, err := app.fetch(*fetcher)
		if err != nil {
			t.Fatalf("error fetching: %v", err)
		}
		assert.Greater(t, len(result), 0)
	}
}

func TestFetcherFetchAndInsertAdapter(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer clean()

	app := testItems.fetcher

	app.initialize(ctx)

	for _, fetcher := range app.Fetchers {
		app.fetchAndInsert(ctx, *fetcher)
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

		err = db.QueryWithoutResult(ctx, "DELETE FROM local_aggregates WHERE name = @name", map[string]any{"name": fetcher.Name})
		if err != nil {
			t.Fatalf("error cleaning up from db: %v", err)
		}

		rdbResult, err := db.Get(ctx, "latestAggregate:"+fetcher.Name)
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

		err = db.Del(ctx, "latestAggregate:"+fetcher.Name)
		if err != nil {
			t.Fatalf("error removing from redis: %v", err)
		}
	}
}
