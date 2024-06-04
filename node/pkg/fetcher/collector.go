package fetcher

import (
	"context"
	"time"

	"bisonai.com/orakl/node/pkg/utils/calculator"
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
	log.Debug().Str("Player", "Collector").Str("collector", c.Name).Msg("collectorJob")
	rawResult, err := c.collect(ctx)
	if err != nil {
		log.Error().Str("Player", "Collector").Err(err).Msg("error in collect")
		return err
	}

	if len(rawResult) == 0 {
		return nil
	}

	aggregated, err := calculator.GetFloatMed(rawResult)
	if err != nil {
		log.Error().Str("Player", "Collector").Err(err).Msg("error in GetFloatMed")
		return err
	}
	err = insertLocalAggregateRdb(ctx, c.ID, aggregated)
	if err != nil {
		log.Error().Str("Player", "Collector").Err(err).Msg("error in insertLocalAggregateRdb")
		return err
	}
	return insertLocalAggregatePgsql(ctx, c.ID, aggregated)
}

func (c *Collector) collect(ctx context.Context) ([]float64, error) {
	feedIds := make([]int32, len(c.Feeds))
	for i, feed := range c.Feeds {
		feedIds[i] = feed.ID
	}
	feedData, err := getLatestFeedData(ctx, feedIds)
	if err != nil {
		log.Error().Str("Player", "Collector").Err(err).Msg("error in getLatestFeedData")
		return nil, err
	}
	result := make([]float64, len(feedData))
	for i, data := range feedData {
		result[i] = data.Value
	}
	return result, nil
}
