package fetcher

import (
	"context"
	"fmt"
	"os"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"github.com/rs/zerolog/log"
)

func NewLocalAggregator(
	config Config,
	feeds []Feed,
	localAggregatesChannel chan *LocalAggregate,
	bus *bus.MessageBus,
	latestFeedDataMap *LatestFeedDataMap) *LocalAggregator {
	return &LocalAggregator{
		Config:                 config,
		Feeds:                  feeds,
		aggregatorCtx:          nil,
		cancel:                 nil,
		bus:                    bus,
		localAggregatesChannel: localAggregatesChannel,
		latestFeedDataMap:      latestFeedDataMap,
	}
}

func (c *LocalAggregator) Run(ctx context.Context) {
	aggregatorCtx, cancel := context.WithCancel(ctx)
	c.aggregatorCtx = aggregatorCtx
	c.cancel = cancel
	c.isRunning = true

	localAggregateIntervalRaw := os.Getenv("LOCAL_AGGREGATE_INTERVAL")
	localAggregateInterval, err := time.ParseDuration(localAggregateIntervalRaw)
	if err != nil {
		localAggregateInterval = DefaultLocalAggregateInterval
	}
	localAggregateFrequency := localAggregateInterval

	ticker := time.NewTicker(localAggregateFrequency)
	go func() {
		for {
			select {
			case <-c.aggregatorCtx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				go func() {
					err := c.Job(c.aggregatorCtx)
					if err != nil {
						log.Error().Str("Player", "LocalAggregator").Err(err).Msg("error in localAggregatorJob")
					}
				}()
			}
		}
	}()
}

func (c *LocalAggregator) Job(ctx context.Context) error {
	feeds, err := c.collect(ctx)
	if err != nil {
		return err
	}

	if len(feeds) == 0 {
		return nil
	}

	return c.processFeeds(ctx, feeds)
}

func (c *LocalAggregator) processFeeds(ctx context.Context, feeds []*FeedData) error {
	if isFXPricePair(c.Name) {
		return c.processFXPricePair(ctx, feeds)
	}
	return c.processVolumeWeightedFeeds(ctx, feeds)
}

func (c *LocalAggregator) processFXPricePair(ctx context.Context, feeds []*FeedData) error {
	median, err := calculateMedian(feeds)
	if err != nil {
		log.Error().Err(err).Str("Player", "LocalAggregator").Msg("error in calculateMedian in localAggregator")
		return err
	}
	return c.streamLocalAggregate(ctx, median)
}

func (c *LocalAggregator) processVolumeWeightedFeeds(ctx context.Context, feeds []*FeedData) error {
	volumeWeightedFeeds, medianFeeds := partitionFeeds(feeds)
	vwap, err := calculateVWAP(volumeWeightedFeeds)
	if err != nil {
		log.Error().Err(err).Str("Player", "LocalAggregator").Msg("error in calculateVWAP in localAggregator")
		return err
	}

	median, err := calculateMedian(medianFeeds)
	if err != nil {
		log.Error().Err(err).Str("Player", "LocalAggregator").Msg("error in calculateMedian in localAggregator")
		return err
	}
	log.Debug().Str("Player", "LocalAggregator").Msg(fmt.Sprintf("VWAP: %f Median: %f", vwap, median))
	aggregated := calculateAggregatedPrice(vwap, median)
	return c.streamLocalAggregate(ctx, aggregated)
}

func partitionFeeds(feeds []*FeedData) ([]*FeedData, []*FeedData) {
	volumeWeightedFeeds := []*FeedData{}
	medianFeeds := []*FeedData{}

	for _, feed := range feeds {
		if feed.Volume > 0 {
			volumeWeightedFeeds = append(volumeWeightedFeeds, feed)
		} else {
			medianFeeds = append(medianFeeds, feed)
		}
	}

	return volumeWeightedFeeds, medianFeeds
}

func calculateAggregatedPrice(valueWeightedAveragePrice, medianPrice float64) float64 {
	if valueWeightedAveragePrice == 0 {
		return medianPrice
	} else if medianPrice == 0 {
		return valueWeightedAveragePrice
	}
	return valueWeightedAveragePrice*(1-DefaultMedianRatio) + medianPrice*DefaultMedianRatio
}

func (c *LocalAggregator) streamLocalAggregate(ctx context.Context, aggregated float64) error {
	if aggregated != 0 {
		localAggregate := &LocalAggregate{
			ConfigID:  c.ID,
			Value:     int64(aggregated),
			Timestamp: time.Now(),
		}

		msg := bus.Message{
			From: bus.FETCHER,
			To:   bus.AGGREGATOR,
			Content: bus.MessageContent{
				Command: bus.STREAM_LOCAL_AGGREGATE,
				Args:    map[string]any{"value": localAggregate},
			},
		}
		defer func() { c.localAggregatesChannel <- localAggregate }()
		return c.bus.Publish(msg)
	}

	return nil
}

func (c *LocalAggregator) collect(ctx context.Context) ([]*FeedData, error) {
	feedIds := make([]int32, len(c.Feeds))
	for i, feed := range c.Feeds {
		feedIds[i] = feed.ID
	}
	return c.latestFeedDataMap.GetLatestFeedData(feedIds)
}
