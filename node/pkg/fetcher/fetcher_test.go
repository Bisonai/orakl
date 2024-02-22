//nolint:all
package fetcher

import (
	"context"
	"encoding/json"
	"testing"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestFetcherInitialize(t *testing.T) {
	ctx := context.Background()
	admin, err := setup()
	if err != nil {
		t.Fatalf("error setting up admin: %v", err)
	}
	cleanup, err := insertSampleData(admin, ctx)
	if err != nil {
		t.Fatalf("error inserting sample data: %v", err)
	}

	defer admin.Shutdown()
	defer cleanup()

	b := bus.NewMessageBus()
	fetcher := New(b)
	fetcher.initialize(ctx)
	assert.Greater(t, len(fetcher.Adapters), 0)
	assert.Greater(t, len(fetcher.Adapters[0].Feeds), 0)
}

func TestFetcherFetch(t *testing.T) {
	ctx := context.Background()
	admin, err := setup()
	if err != nil {
		t.Fatalf("error setting up admin: %v", err)
	}
	cleanup, err := insertSampleData(admin, ctx)
	if err != nil {
		t.Fatalf("error inserting sample data: %v", err)
	}

	defer admin.Shutdown()
	defer cleanup()

	b := bus.NewMessageBus()
	fetcher := New(b)
	fetcher.initialize(ctx)
	result, err := fetcher.fetch(fetcher.Adapters[0])
	if err != nil {
		t.Fatalf("error fetching: %v", err)
	}
	assert.Greater(t, len(result), 0)
}

func TestFetcherRunAdapter(t *testing.T) {
	ctx := context.Background()
	admin, err := setup()
	if err != nil {
		t.Fatalf("error setting up admin: %v", err)
	}
	cleanup, err := insertSampleData(admin, ctx)
	if err != nil {
		t.Fatalf("error inserting sample data: %v", err)
	}

	defer admin.Shutdown()
	defer cleanup()

	b := bus.NewMessageBus()
	fetcher := New(b)
	fetcher.initialize(ctx)
	err = fetcher.fetchAll(ctx)
	if err != nil {
		t.Fatalf("error running adapter: %v", err)
	}

	// read aggregate from db
	pgResult, err := db.QueryRow[Aggregate](ctx, "SELECT * FROM local_aggregates WHERE name = @name", map[string]any{"name": fetcher.Adapters[0].Name})
	if err != nil {
		t.Fatalf("error reading from db: %v", err)
	}
	assert.NotNil(t, pgResult)

	// cleanup aggregate from db
	err = db.QueryWithoutResult(ctx, "DELETE FROM local_aggregates WHERE name = @name", map[string]any{"name": fetcher.Adapters[0].Name})
	if err != nil {
		t.Fatalf("error cleaning up from db: %v", err)
	}

	// read aggregate from redis
	rdbResult, err := db.Get(ctx, "latestAggregate:"+fetcher.Adapters[0].Name)
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

	// remove aggregate from redis
	err = db.Del(ctx, "latestAggregate:"+fetcher.Adapters[0].Name)
	if err != nil {
		t.Fatalf("error removing from redis: %v", err)
	}
}
