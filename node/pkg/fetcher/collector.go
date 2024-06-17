package fetcher

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

func NewCollector(config Config, feeds []Feed) *Collector {
	return &Collector{
		Config:       config,
		Feeds:        feeds,
		collectorCtx: nil,
		cancel:       nil,
	}
}

func (c *Collector) Run(ctx context.Context) {
	collectorCtx, cancel := context.WithCancel(ctx)
	c.collectorCtx = collectorCtx
	c.cancel = cancel
	c.isRunning = true

	collectorFrequency := time.Duration(c.FetchInterval) * time.Millisecond
	ticker := time.NewTicker(collectorFrequency)
	go func() {
		for {
			select {
			case <-c.collectorCtx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				err := c.Job(c.collectorCtx)
				if err != nil {
					log.Error().Str("Player", "Collector").Err(err).Msg("error in collectorJob")
				}
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

	if isFXPricePair(c.Name) {
		median, medianErr := calculateMedian(feeds)
		if medianErr != nil {
			return medianErr
		}
		return insertAggregateData(ctx, c.ID, median)
	}

	volumeWeightedFeeds := filterFeedsWithVolume(feeds)
	vwap, err := calculateVWAP(volumeWeightedFeeds)
	if err != nil {
		return err
	}
	median, err := calculateMedian(feeds)
	if err != nil {
		return err
	}

	var aggregated float64
	if vwap != 0 && median != 0 {
		aggregated = calculateAggregatedPrice(vwap, median)
	} else if vwap == 0 {
		aggregated = median
	} else {
		aggregated = vwap
	}

	return insertAggregateData(ctx, c.ID, aggregated)
}

func filterFeedsWithVolume(feeds []FeedData) []FeedData {
	volumeWeightedFeeds := []FeedData{}
	for _, feed := range feeds {
		if feed.Volume > 0 {
			volumeWeightedFeeds = append(volumeWeightedFeeds, feed)
		}
	}
	return volumeWeightedFeeds
}

func calculateAggregatedPrice(valueWeightedAveragePrice, medianPrice float64) float64 {
	return valueWeightedAveragePrice*(1-DefaultMedianRatio) + medianPrice*DefaultMedianRatio
}

func insertAggregateData(ctx context.Context, id int32, aggregated float64) error {
	if aggregated == 0 {
		return nil
	}
	err1 := insertLocalAggregateRdb(ctx, id, aggregated)
	err2 := insertLocalAggregatePgsql(ctx, id, aggregated)

	var errs []error
	if err1 != nil {
		errs = append(errs, err1)
	}
	if err2 != nil {
		errs = append(errs, err2)
	}
	if len(errs) > 0 {
		return fmt.Errorf("%v", errs)
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
