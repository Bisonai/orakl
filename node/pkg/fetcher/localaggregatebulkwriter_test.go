package fetcher

import (
	"context"
	"math"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/db"
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

	configMap := make(map[int32]Config, len(configs))
	for _, config := range configs {
		configMap[config.ID] = config
	}

	localAggregatesChannel := make(chan *LocalAggregate, LocalAggregatesChannelSize)
	app.LocalAggregators = make(map[int32]*LocalAggregator, len(configs))
	app.LocalAggregateBulkWriter = NewLocalAggregateBulkWriter(DefaultLocalAggregateInterval)
	app.LocalAggregateBulkWriter.localAggregatesChannel = localAggregatesChannel

	feedData := []*FeedData{}
	for _, config := range configs {
		localAggregatorFeeds, getFeedsErr := app.getFeeds(ctx, config.ID)
		if getFeedsErr != nil {
			t.Fatalf("error getting configs: %v", getFeedsErr)
		}
		app.LocalAggregators[config.ID] = NewLocalAggregator(config, localAggregatorFeeds, localAggregatesChannel, testItems.messageBus, app.LatestFeedDataMap)
		for _, feed := range localAggregatorFeeds {
			feedData = append(feedData, &FeedData{FeedID: feed.ID, Value: DUMMY_FEED_VALUE, Timestamp: nil, Volume: DUMMY_FEED_VALUE})
		}
	}
	err = app.startAllLocalAggregators(ctx)
	if err != nil {
		t.Fatalf("error starting localAggregators: %v", err)
	}

	err = app.LatestFeedDataMap.SetLatestFeedData(feedData)
	if err != nil {
		t.Fatalf("error setting latest feed data: %v", err)
	}

	data := <-localAggregatesChannel

	decimals := configMap[data.ConfigID].Decimals
	if decimals == nil {
		defaultDecimals := DECIMALS
		decimals = &defaultDecimals
	}

	assert.Equal(t, DUMMY_FEED_VALUE*math.Pow10(*decimals), float64(data.Value))

	go app.LocalAggregateBulkWriter.Run(ctx)

	time.Sleep(DefaultLocalAggregateInterval * 4)

	pgsqlData, pgsqlErr := db.QueryRow[LocalAggregate](ctx, "SELECT * FROM local_aggregates WHERE config_id = @config_id", map[string]any{
		"config_id": data.ConfigID,
	})
	if pgsqlErr != nil {
		t.Fatalf("error getting local aggregate from pgsql: %v", pgsqlErr)
	}
	assert.Equal(t, DUMMY_FEED_VALUE*math.Pow10(*decimals), float64(pgsqlData.Value))
}
