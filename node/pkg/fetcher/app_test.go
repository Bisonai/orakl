//nolint:all
package fetcher

import (
	"context"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/db"
	"github.com/stretchr/testify/assert"
)

const WAIT_SECONDS = 4 * time.Second
const DUMMY_FEED_VALUE = 1.0

func TestFetcherInitialize(t *testing.T) {
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

	assert.Greater(t, len(app.Fetchers), 0)
	for _, adapter := range app.Fetchers {
		assert.Greater(t, len(adapter.Feeds), 0)
	}
}

func TestAppRun(t *testing.T) {
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

	err = app.Run(ctx)
	if err != nil {
		t.Fatalf("error running fetcher: %v", err)
	}
	for _, fetcher := range app.Fetchers {
		assert.True(t, fetcher.isRunning)
	}
	for _, localAggregator := range app.LocalAggregators {
		assert.True(t, localAggregator.isRunning)
	}
	assert.True(t, app.FeedDataBulkWriter.isRunning)

	time.Sleep(WAIT_SECONDS)

	err = app.stopAll(ctx)
	if err != nil {
		t.Fatalf("error stopping fetcher: %v", err)
	}
	for _, fetcher := range app.Fetchers {
		assert.False(t, fetcher.isRunning)
	}
	for _, localAggregator := range app.LocalAggregators {
		assert.False(t, localAggregator.isRunning)
	}
	assert.False(t, app.FeedDataBulkWriter.isRunning)

	for _, fetcher := range app.Fetchers {
		for _, feed := range fetcher.Feeds {

			result, letestFeedDataErr := app.LatestFeedDataMap.GetLatestFeedData([]int32{feed.ID})
			if letestFeedDataErr != nil {
				t.Fatalf("error getting latest feed data: %v", letestFeedDataErr)
			}
			assert.NotNil(t, result)
		}
		data := <-testItems.aggChan
		assert.NotNil(t, data)
	}

	err = app.FeedDataBulkWriter.Job(ctx)
	if err != nil {
		t.Fatalf("error running FeedDataBulkWriter job: %v", err)
	}

	feedResult, err := db.QueryRows[FeedData](ctx, "SELECT * FROM feed_data", nil)
	if err != nil {
		t.Fatalf("error querying feed data: %v", err)
	}
	assert.Greater(t, len(feedResult), 0)

	localAggregateResult, err := db.QueryRows[LocalAggregate](ctx, "SELECT * FROM local_aggregates", nil)
	if err != nil {
		t.Fatalf("error querying local aggregates: %v", err)
	}
	assert.Greater(t, len(localAggregateResult), 0)
}
