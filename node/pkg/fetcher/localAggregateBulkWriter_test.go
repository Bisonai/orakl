package fetcher

import (
	"context"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/common/keys"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestLocalAggregateBulkWriter(t *testing.T) {
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

	// get configs, initialize channel, and start localAggregators
	configs, err := app.getConfigs(ctx)
	if err != nil {
		t.Fatalf("error getting configs: %v", err)
	}

	localAggregatesChannel := make(chan *LocalAggregate, LocalAggregatesChannelSize)
	app.LocalAggregators = make(map[int32]*LocalAggregator, len(configs))
	app.LocalAggregateBulkWriter = NewLocalAggregateBulkWriter(DefaultLocalAggregateInterval)
	app.LocalAggregateBulkWriter.localAggregatesChannel = localAggregatesChannel

	feedData := make(map[string]any)
	for _, config := range configs {
		localAggregatorFeeds, getFeedsErr := app.getFeeds(ctx, config.ID)
		if getFeedsErr != nil {
			t.Fatalf("error getting configs: %v", getFeedsErr)
		}
		app.LocalAggregators[config.ID] = NewLocalAggregator(config, localAggregatorFeeds, localAggregatesChannel, testItems.messageBus)
		for _, feed := range localAggregatorFeeds {
			feedData[keys.LatestFeedDataKey(feed.ID)] = FeedData{FeedID: feed.ID, Value: DUMMY_FEED_VALUE, Timestamp: nil, Volume: DUMMY_FEED_VALUE}
		}
	}
	err = app.startAllLocalAggregators(ctx)
	if err != nil {
		t.Fatalf("error starting localAggregators: %v", err)
	}

	err = db.MSetObject(ctx, feedData)
	if err != nil {
		t.Fatalf("error setting feed data in redis: %v", err)
	}

	data := <-localAggregatesChannel
	assert.Equal(t, DUMMY_FEED_VALUE, float64(data.Value))

	go app.LocalAggregateBulkWriter.Run(ctx)

	time.Sleep(DefaultLocalAggregateInterval * 4)

	pgsqlData, pgsqlErr := db.QueryRow[LocalAggregate](ctx, "SELECT * FROM local_aggregates WHERE config_id = @config_id", map[string]any{
		"config_id": data.ConfigID,
	})
	if pgsqlErr != nil {
		t.Fatalf("error getting local aggregate from pgsql: %v", pgsqlErr)
	}
	assert.Equal(t, float64(pgsqlData.Value), DUMMY_FEED_VALUE)
}
