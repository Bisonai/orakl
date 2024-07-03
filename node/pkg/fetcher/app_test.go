//nolint:all
package fetcher

import (
	"context"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/common/keys"
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
	for _, collector := range app.Collectors {
		assert.True(t, collector.isRunning)
	}
	assert.True(t, app.Streamer.isRunning)

	time.Sleep(WAIT_SECONDS)

	err = app.stopAll(ctx)
	if err != nil {
		t.Fatalf("error stopping fetcher: %v", err)
	}
	for _, fetcher := range app.Fetchers {
		assert.False(t, fetcher.isRunning)
	}
	for _, collector := range app.Collectors {
		assert.False(t, collector.isRunning)
	}
	assert.False(t, app.Streamer.isRunning)

	for _, fetcher := range app.Fetchers {
		for _, feed := range fetcher.Feeds {

			result, letestFeedDataErr := db.GetObject[FeedData](ctx, keys.LatestFeedDataKey(feed.ID))
			if letestFeedDataErr != nil {
				t.Fatalf("error getting latest feed data: %v", letestFeedDataErr)
			}
			assert.NotNil(t, result)
		}
		rdbResult, localAggregateErr := db.Get(ctx, keys.LocalAggregateKey(fetcher.Config.ID))
		if localAggregateErr != nil {
			t.Fatalf("error getting local aggregate: %v", localAggregateErr)
		}
		assert.NotNil(t, rdbResult)
	}

	err = app.Streamer.Job(ctx)
	if err != nil {
		t.Fatalf("error running streamer job: %v", err)
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

func TestLocalAggregatesChannel(t *testing.T) {
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

	// get configs, initialize channel, and start collectors
	configs, err := app.getConfigs(ctx)
	// t.Logf("configs: %v", configs)
	if err != nil {
		t.Fatalf("error getting configs: %v", err)
	}
	configsLen := len(configs)

	localAggregatesChannel := make(chan LocalAggregate, LocalAggregatesChannelSize)
	app.Collectors = make(map[int32]*Collector, len(configs))

	feedData := make(map[string]any)
	for _, config := range configs {
		collectorFeeds, getFeedsErr := app.getFeeds(ctx, config.ID)
		// t.Logf("collectorFeeds: %v", collectorFeeds)
		if getFeedsErr != nil {
			t.Fatalf("error getting configs: %v", getFeedsErr)
		}
		app.Collectors[config.ID] = NewCollector(config, collectorFeeds, localAggregatesChannel)
		for i, feed := range collectorFeeds {
			feedData[keys.LatestFeedDataKey(feed.ID)] = FeedData{FeedID: feed.ID, Value: float64(i+1), Timestamp: nil, Volume: float64(i+1)}
		}
	}
	err = app.startAllCollectors(ctx)
	if err != nil {
		t.Fatalf("error starting collectors: %v", err)
	}

	err = db.MSetObject(ctx, feedData)
	if err != nil {
		t.Fatalf("error setting feed data in redis: %v", err)
	}

	// wait until collectors fetch and process data from db
	time.Sleep(250*time.Millisecond)
	assert.Equal(t, len(localAggregatesChannel), configsLen)

	go app.localAggregatesChannelProcessor(ctx, localAggregatesChannel)
	time.Sleep(250*time.Millisecond)
	assert.Equal(t, len(localAggregatesChannel), 0)
}