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

	fetcher := testItems.fetcher

	err = fetcher.initialize(ctx)
	if err != nil {
		t.Fatalf("error initializing fetcher: %v", err)
	}

	assert.Greater(t, len(fetcher.Adapters), 0)
	for _, adapter := range fetcher.Adapters {
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

	fetcher := testItems.fetcher

	err = fetcher.initialize(ctx)
	if err != nil {
		t.Fatalf("error initializing fetcher: %v", err)
	}

	rowsBefore, err := db.QueryRows[Aggregate](ctx, "SELECT * FROM local_aggregates", nil)
	if err != nil {
		t.Fatalf("error reading from db: %v", err)
	}
	assert.Equal(t, 0, len(rowsBefore))

	err = fetcher.Run(ctx)
	if err != nil {
		t.Fatalf("error running fetcher: %v", err)
	}

	for _, adapter := range fetcher.Adapters {
		assert.True(t, adapter.isRunning)
	}

	// wait for fetcher to run
	time.Sleep(WAIT_SECONDS)

	// stop running after 2 seconds
	for _, adapter := range fetcher.Adapters {
		fetcher.stopAdapter(ctx, adapter)
		assert.False(t, adapter.isRunning)
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

	for _, adapter := range fetcher.Adapters {
		rdbResult, err := db.Get(ctx, "latestAggregate:"+adapter.Name)
		if err != nil {
			t.Fatalf("error reading from redis: %v", err)
		}
		assert.NotNil(t, rdbResult)

		err = db.Del(ctx, "latestAggregate:"+adapter.Name)
		if err != nil {
			t.Fatalf("error removing from redis: %v", err)
		}
	}
}

func TestFetcherAdapterStart(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer clean()

	fetcher := testItems.fetcher

	err = fetcher.initialize(ctx)
	if err != nil {
		t.Fatalf("error initializing fetcher: %v", err)
	}

	rowsBefore, err := db.QueryRows[Aggregate](ctx, "SELECT * FROM local_aggregates", nil)
	if err != nil {
		t.Fatalf("error reading from db: %v", err)
	}
	assert.Equal(t, 0, len(rowsBefore))

	for _, adapter := range fetcher.Adapters {
		err = fetcher.startAdapter(ctx, adapter)
		if err != nil {
			t.Fatalf("error starting adapter: %v", err)
		}
		assert.True(t, adapter.isRunning)
	}

	// wait for fetcher to run
	time.Sleep(WAIT_SECONDS)

	// stop running after 2 seconds
	for _, adapter := range fetcher.Adapters {
		fetcher.stopAdapter(ctx, adapter)
		assert.False(t, adapter.isRunning)
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
	for _, adapter := range fetcher.Adapters {
		rdbResult, err := db.Get(ctx, "latestAggregate:"+adapter.Name)
		if err != nil {
			t.Fatalf("error reading from redis: %v", err)
		}
		assert.NotNil(t, rdbResult)

		err = db.Del(ctx, "latestAggregate:"+adapter.Name)
		if err != nil {
			t.Fatalf("error removing from redis: %v", err)
		}
	}
}

func TestFetcherAdapterStop(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer clean()

	fetcher := testItems.fetcher

	err = fetcher.initialize(ctx)
	if err != nil {
		t.Fatalf("error initializing fetcher: %v", err)
	}

	// first start adapters to stop
	for _, adapter := range fetcher.Adapters {
		err = fetcher.startAdapter(ctx, adapter)
		if err != nil {
			t.Fatalf("error starting adapter: %v", err)
		}
		assert.True(t, adapter.isRunning)
	}

	// wait for fetcher to run
	time.Sleep(WAIT_SECONDS)

	// stop adapters
	for _, adapter := range fetcher.Adapters {
		fetcher.stopAdapter(ctx, adapter)
		assert.False(t, adapter.isRunning)
	}

	rowsBefore, err := db.QueryRows[Aggregate](ctx, "SELECT * FROM local_aggregates", nil)
	if err != nil {
		t.Fatalf("error reading from db: %v", err)
	}
	assert.Greater(t, len(rowsBefore), 0)

	time.Sleep(WAIT_SECONDS)

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
	for _, adapter := range fetcher.Adapters {
		rdbResult, err := db.Get(ctx, "latestAggregate:"+adapter.Name)
		if err != nil {
			t.Fatalf("error reading from redis: %v", err)
		}
		assert.NotNil(t, rdbResult)

		err = db.Del(ctx, "latestAggregate:"+adapter.Name)
		if err != nil {
			t.Fatalf("error removing from redis: %v", err)
		}
	}
}

func TestFetcherAdapterStartById(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer clean()

	fetcher := testItems.fetcher

	err = fetcher.initialize(ctx)
	if err != nil {
		t.Fatalf("error initializing fetcher: %v", err)
	}

	fetcher.subscribe(ctx)

	rowsBefore, err := db.QueryRows[Aggregate](ctx, "SELECT * FROM local_aggregates", nil)
	if err != nil {
		t.Fatalf("error reading from db: %v", err)
	}
	assert.Equal(t, 0, len(rowsBefore))

	for _, adapter := range fetcher.Adapters {
		result, _err := tests.PostRequest[Adapter](testItems.app, "/api/v1/adapter/activate/"+strconv.FormatInt(adapter.ID, 10), nil)
		if _err != nil {
			t.Fatalf("error starting adapter: %v", _err)
		}

		assert.True(t, result.Active)
	}

	time.Sleep(WAIT_SECONDS)

	for _, adapter := range fetcher.Adapters {
		assert.True(t, adapter.isRunning)
		fetcher.stopAdapter(ctx, adapter)
		assert.False(t, adapter.isRunning)
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
	for _, adapter := range fetcher.Adapters {
		rdbResult, err := db.Get(ctx, "latestAggregate:"+adapter.Name)
		if err != nil {
			t.Fatalf("error reading from redis: %v", err)
		}
		assert.NotNil(t, rdbResult)

		err = db.Del(ctx, "latestAggregate:"+adapter.Name)
		if err != nil {
			t.Fatalf("error removing from redis: %v", err)
		}
	}
}

func TestFetcherAdapterStopById(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer clean()

	fetcher := testItems.fetcher

	err = fetcher.initialize(ctx)
	if err != nil {
		t.Fatalf("error initializing fetcher: %v", err)
	}

	err = fetcher.Run(ctx)
	if err != nil {
		t.Fatalf("error running fetcher: %v", err)
	}
	for _, adapter := range fetcher.Adapters {
		assert.True(t, adapter.isRunning)
	}

	time.Sleep(WAIT_SECONDS)
	rowsBefore, err := db.QueryRows[Aggregate](ctx, "SELECT * FROM local_aggregates", nil)
	if err != nil {
		t.Fatalf("error reading from db: %v", err)
	}
	assert.Greater(t, len(rowsBefore), 0)

	for _, adapter := range fetcher.Adapters {
		result, _err := tests.PostRequest[Adapter](testItems.app, "/api/v1/adapter/deactivate/"+strconv.FormatInt(adapter.ID, 10), nil)
		if _err != nil {
			t.Fatalf("error stopping adapter: %v", _err)
		}

		assert.False(t, result.Active)
	}

	time.Sleep(WAIT_SECONDS)
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
	for _, adapter := range fetcher.Adapters {
		rdbResult, err := db.Get(ctx, "latestAggregate:"+adapter.Name)
		if err != nil {
			t.Fatalf("error reading from redis: %v", err)
		}
		assert.NotNil(t, rdbResult)

		err = db.Del(ctx, "latestAggregate:"+adapter.Name)
		if err != nil {
			t.Fatalf("error removing from redis: %v", err)
		}
	}

}

func TestFetcherFetch(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer clean()

	fetcher := testItems.fetcher

	err = fetcher.initialize(ctx)
	if err != nil {
		t.Fatalf("error initializing fetcher: %v", err)
	}

	for _, adapter := range fetcher.Adapters {
		result, err := fetcher.fetch(*adapter)
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

	fetcher := testItems.fetcher

	fetcher.initialize(ctx)

	for _, adapter := range fetcher.Adapters {
		fetcher.fetchAndInsert(ctx, *adapter)
	}
	if err != nil {
		t.Fatalf("error running adapter: %v", err)
	}

	for _, adapter := range fetcher.Adapters {
		pgResult, err := db.QueryRow[Aggregate](ctx, "SELECT * FROM local_aggregates WHERE name = @name", map[string]any{"name": adapter.Name})
		if err != nil {
			t.Fatalf("error reading from db: %v", err)
		}
		assert.NotNil(t, pgResult)

		err = db.QueryWithoutResult(ctx, "DELETE FROM local_aggregates WHERE name = @name", map[string]any{"name": adapter.Name})
		if err != nil {
			t.Fatalf("error cleaning up from db: %v", err)
		}

		rdbResult, err := db.Get(ctx, "latestAggregate:"+adapter.Name)
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

		err = db.Del(ctx, "latestAggregate:"+adapter.Name)
		if err != nil {
			t.Fatalf("error removing from redis: %v", err)
		}
	}
}
