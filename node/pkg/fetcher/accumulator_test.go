package fetcher

import (
	"context"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/common/keys"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestAccumulator(t *testing.T) {
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
	if err != nil {
		t.Fatalf("error getting configs: %v", err)
	}

	localAggregatesChannel := make(chan LocalAggregate, LocalAggregatesChannelSize)
	app.Collectors = make(map[int32]*Collector, len(configs))
	app.Accumulator = NewAccumulator(DefaultLocalAggregateInterval)
	app.Accumulator.accumulatorChannel = localAggregatesChannel

	feedData := make(map[string]any)
	for _, config := range configs {
		collectorFeeds, getFeedsErr := app.getFeeds(ctx, config.ID)
		if getFeedsErr != nil {
			t.Fatalf("error getting configs: %v", getFeedsErr)
		}
		app.Collectors[config.ID] = NewCollector(config, collectorFeeds, localAggregatesChannel)
		for _, feed := range collectorFeeds {
			feedData[keys.LatestFeedDataKey(feed.ID)] = FeedData{FeedID: feed.ID, Value: DUMMY_FEED_VALUE, Timestamp: nil, Volume: DUMMY_FEED_VALUE}
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

	data := <-localAggregatesChannel
	assert.Equal(t, float64(data.Value), DUMMY_FEED_VALUE)

	go app.Accumulator.Run(ctx)

	time.Sleep(DefaultLocalAggregateInterval * 4)

	redisData, redisErr := db.GetObject[LocalAggregate](ctx, keys.LocalAggregateKey(data.ConfigID))
	if redisErr != nil {
		t.Fatalf("error getting local aggregate from redis: %v", redisErr)
	}
	assert.Equal(t, float64(redisData.Value), DUMMY_FEED_VALUE)

	pgsqlData, pgsqlErr := db.QueryRow[LocalAggregate](ctx, "SELECT * FROM local_aggregates WHERE config_id = @config_id", map[string]any{
		"config_id": data.ConfigID,
	})
	if pgsqlErr != nil {
		t.Fatalf("error getting local aggregate from pgsql: %v", pgsqlErr)
	}
	assert.Equal(t, float64(pgsqlData.Value), DUMMY_FEED_VALUE)
}
