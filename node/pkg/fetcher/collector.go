package fetcher

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

func NewCollector(config Config, feeds []Feed, localAggregatesChannel chan LocalAggregate) *Collector {
	return &Collector{
		Config:                 config,
		Feeds:                  feeds,
		collectorCtx:           nil,
		cancel:                 nil,
		localAggregatesChannel: localAggregatesChannel,
	}
}

func (c *Collector) Run(ctx context.Context) {
	collectorCtx, cancel := context.WithCancel(ctx)
	c.collectorCtx = collectorCtx
	c.cancel = cancel
	c.isRunning = true

	localAggregateIntervalRaw := os.Getenv("LOCAL_AGGREGATE_INTERVAL")
	localAggregateInterval, err := time.ParseDuration(localAggregateIntervalRaw)
	if err != nil {
		log.Warn().Str("Player", "Collector").Err(err).Msg("error in ParseDuration in collector, using default")
		localAggregateInterval = DefaultLocalAggregateInterval
	}
	collectorFrequency := localAggregateInterval

	ticker := time.NewTicker(collectorFrequency)
	go func() {
		for {
			select {
			case <-c.collectorCtx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				go func() {
					err := c.Job(c.collectorCtx)
					if err != nil {
						log.Error().Str("Player", "Collector").Err(err).Msg("error in collectorJob")
					}
				}()
			}
		}
	}()
}

func (c *Collector) Job(ctx context.Context) error {
	feeds, err := c.collect(ctx)
	if err != nil {
		return err
	}

	if len(feeds) == 0 {
		return nil
	}

	return c.processFeeds(ctx, feeds)
}

func (c *Collector) processFeeds(ctx context.Context, feeds []FeedData) error {
	if isFXPricePair(c.Name) {
		return c.processFXPricePair(ctx, feeds)
	}
	return c.processVolumeWeightedFeeds(ctx, feeds)
}

func (c *Collector) processFXPricePair(ctx context.Context, feeds []FeedData) error {
	median, err := calculateMedian(feeds)
	if err != nil {
		log.Error().Err(err).Str("Player", "Collector").Msg("error in calculateMedian in collector")
		return err
	}
	return c.streamLocalAggregate(ctx, median)
}

func (c *Collector) processVolumeWeightedFeeds(ctx context.Context, feeds []FeedData) error {
	volumeWeightedFeeds, medianFeeds := partitionFeeds(feeds)
	vwap, err := calculateVWAP(volumeWeightedFeeds)
	if err != nil {
		log.Error().Err(err).Str("Player", "Collector").Msg("error in calculateVWAP in collector")
		return err
	}

	median, err := calculateMedian(medianFeeds)
	if err != nil {
		log.Error().Err(err).Str("Player", "Collector").Msg("error in calculateMedian in collector")
		return err
	}
	log.Debug().Str("Player", "Collector").Msg(fmt.Sprintf("VWAP: %f Median: %f", vwap, median))
	aggregated := calculateAggregatedPrice(vwap, median)
	return c.streamLocalAggregate(ctx, aggregated)
}

func partitionFeeds(feeds []FeedData) ([]FeedData, []FeedData) {
	volumeWeightedFeeds := []FeedData{}
	medianFeeds := []FeedData{}

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

func (c *Collector) streamLocalAggregate(ctx context.Context, aggregated float64) error {
	if aggregated != 0 {
		c.localAggregatesChannel <- LocalAggregate{
			ConfigID:  c.ID,
			Value:     int64(aggregated),
			Timestamp: time.Now(),
		}
	}

	return nil
}

func (c *Collector) collect(ctx context.Context) ([]FeedData, error) {
	feedIds := make([]int32, len(c.Feeds))
	for i, feed := range c.Feeds {
		feedIds[i] = feed.ID
	}
	return getLatestFeedData(ctx, feedIds)
}
