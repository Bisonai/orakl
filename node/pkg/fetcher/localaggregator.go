package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"slices"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/metrics"
	"github.com/montanaflynn/stats"
	"github.com/rs/zerolog/log"
)

func NewLocalAggregator(
	config Config,
	feeds []Feed,
	bus *bus.MessageBus,
	latestFeedDataMap *LatestFeedDataMap,
	localAggregateValueMap *LocalAggregateValueMap) *LocalAggregator {
	return &LocalAggregator{
		Config:                 config,
		Feeds:                  feeds,
		cancel:                 nil,
		bus:                    bus,
		latestFeedDataMap:      latestFeedDataMap,
		localAggregateValueMap: localAggregateValueMap,
	}
}

func (c *LocalAggregator) Run(ctx context.Context) {
	aggregatorCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.isRunning = true

	localAggregateIntervalRaw := os.Getenv("LOCAL_AGGREGATE_INTERVAL")
	localAggregateInterval, err := time.ParseDuration(localAggregateIntervalRaw)
	if err != nil {
		localAggregateInterval = DefaultLocalAggregateInterval
	}
	localAggregateFrequency := localAggregateInterval

	ticker := time.NewTicker(localAggregateFrequency)
	// The loop must watch only its own aggregatorCtx (captured here), so a
	// quick STOP->START can't keep an old loop alive.
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-aggregatorCtx.Done():
				return
			case <-ticker.C:
				// select picks randomly when a tick and cancel are both ready;
				// don't start a new job after STOP.
				if aggregatorCtx.Err() != nil {
					return
				}
				go func() {
					err := c.Job(aggregatorCtx)
					if err != nil {
						log.Error().Str("Player", "LocalAggregator").Err(err).Msg("error in localAggregatorJob")
					}
				}()
			}
		}
	}()
}

func (c *LocalAggregator) Job(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}

	feeds, err := c.collect(ctx)
	if err != nil {
		return err
	}

	if len(feeds) < len(c.Feeds)/2 {
		log.Debug().Str("Player", "LocalAggregator").Msg("not enough feeds")
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
	filtered, err := filterOutliers(feeds)
	if err != nil {
		log.Error().Err(err).Str("Player", "LocalAggregator").Msg("error in filterOutliers in localAggregator")
		return err
	}

	volumeWeightedFeeds, medianFeeds := partitionFeeds(filtered)
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

func filterOutliers(feeds []*FeedData) ([]*FeedData, error) {
	if len(feeds) < 5 {
		return feeds, nil
	}

	data := make([]float64, len(feeds))
	for i, feed := range feeds {
		data[i] = feed.Value
	}

	outliers, err := stats.QuartileOutliers(data)
	if err != nil {
		return nil, err
	}

	if outliers.Mild.Len() == 0 && outliers.Extreme.Len() == 0 {
		return feeds, nil
	}

	median, err := stats.Median(data)
	if err != nil {
		return nil, err
	}

	maxOutliersToRemove := int(float64(len(feeds)) * MaxOutlierRemovalRatio)

	filtered := feeds
	var extremes stats.Float64Data
	if outliers.Extreme.Len() > 0 {
		slices.SortFunc(outliers.Extreme, func(a, b float64) int {
			if math.Abs(median-a) < math.Abs(median-b) {
				return 1
			} else if math.Abs(median-a) > math.Abs(median-b) {
				return -1
			} else {
				return 0
			}
		})

		extremes = outliers.Extreme[:min(maxOutliersToRemove, outliers.Extreme.Len())]
		filtered = slices.DeleteFunc(feeds, func(feed *FeedData) bool {
			return slices.Contains(extremes, feed.Value)
		})

		log.Debug().Int32("feed", feeds[0].FeedID).Any("extremes", extremes).Msg("extremes")
	}

	if extremes.Len() < maxOutliersToRemove && outliers.Mild.Len() > 0 {
		slices.SortFunc(outliers.Mild, func(a, b float64) int {
			if math.Abs(median-a) < math.Abs(median-b) {
				return 1
			} else if math.Abs(median-a) > math.Abs(median-b) {
				return -1
			} else {
				return 0
			}
		})

		milds := outliers.Mild[:min(maxOutliersToRemove-extremes.Len(), outliers.Mild.Len())]
		filtered = slices.DeleteFunc(filtered, func(feed *FeedData) bool {
			return slices.Contains(milds, feed.Value)
		})

		log.Debug().Int32("feed", feeds[0].FeedID).Any("milds", milds).Msg("milds")
	}

	return filtered, nil
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
	if aggregated == 0 {
		return nil
	}

	// Synthetic configs (e.g. KAIA-KRW = KAIA-USDT / KRW-USD with
	// MultiplyByReciprocal, since KRW-USD is USD per KRW) source their
	// multiplier from another config's most recent raw aggregate.  If the
	// source isn't available yet we skip emitting this round rather than
	// publishing a zeroed-out value.
	if c.MultiplyBy != nil && *c.MultiplyBy != "" {
		entry, ok := c.localAggregateValueMap.Get(*c.MultiplyBy)
		if !ok || entry.Value == 0 {
			log.Debug().Str("Player", "LocalAggregator").
				Str("config", c.Name).
				Str("multiplyBy", *c.MultiplyBy).
				Msg("multiplier source not yet available, skipping")
			return nil
		}
		// Reject obviously stale multipliers using this config's freshness
		// budget; if it isn't set, fall back to the freshness used elsewhere.
		freshness := time.Minute
		if c.Config.FeedDataFreshness != nil && *c.Config.FeedDataFreshness > 0 {
			freshness = time.Duration(*c.Config.FeedDataFreshness) * time.Millisecond
		}
		if time.Since(entry.Timestamp) > freshness {
			log.Warn().Str("Player", "LocalAggregator").
				Str("config", c.Name).
				Str("multiplyBy", *c.MultiplyBy).
				Dur("age", time.Since(entry.Timestamp)).
				Dur("freshness", freshness).
				Msg("multiplier source is stale, skipping")
			return nil
		}
		// Some price configs (e.g. KRW-USD) intentionally store the
		// reciprocal of the natural rate.  When the synthetic config
		// needs the natural direction it sets MultiplyByReciprocal so
		// we divide by the source instead of multiplying.
		if c.MultiplyByReciprocal {
			aggregated /= entry.Value
		} else {
			aggregated *= entry.Value
		}
	}

	// Don't cache or publish from a job that was still running when
	// STOP/REFRESH cancelled its loop.
	if ctx.Err() != nil {
		return nil
	}

	// Cache our own raw value so other configs can multiply against us.
	if c.localAggregateValueMap != nil {
		c.localAggregateValueMap.Set(c.Name, aggregated)
	}

	localAggregate := &LocalAggregate{
		ConfigID:  c.ID,
		Value:     c.applyDecimals(aggregated),
		Timestamp: time.Now(),
	}
	metrics.SetLocalAggregate(c.Name, localAggregate.Value, localAggregate.Timestamp)

	msg := bus.Message{
		From: bus.FETCHER,
		To:   bus.AGGREGATOR,
		Content: bus.MessageContent{
			Command: bus.STREAM_LOCAL_AGGREGATE,
			Args:    map[string]any{"value": localAggregate},
		},
	}
	return c.bus.Publish(msg)
}

// applyDecimals scales the aggregate to the feed's integer representation,
// rounding to the nearest unit. Truncating would bias every value down by up
// to one unit (0.018% for IDR-USD at 8 decimals) and turn float products such
// as 1.31025*1e8 = 131024999.99999999 into 131024999.
func (c *LocalAggregator) applyDecimals(value float64) int64 {
	if c.Config.Decimals != nil && *c.Config.Decimals != 0 {
		return int64(math.Round(value * math.Pow10(*c.Config.Decimals)))
	}
	return int64(math.Round(value * math.Pow10(DECIMALS)))
}

func (c *LocalAggregator) collect(ctx context.Context) ([]*FeedData, error) {
	feedIds := make([]int32, len(c.Feeds))
	for i, feed := range c.Feeds {
		feedIds[i] = feed.ID
	}

	feeds, err := c.latestFeedDataMap.GetLatestFeedData(feedIds)
	if err != nil {
		return nil, err
	}

	feeds = c.selectMidPrices(feeds)
	feeds = c.applyPerFeedMultipliers(feeds)
	return c.filterStaleFeeds(feeds), nil
}

// selectMidPrices, for configs with priceSource "mid", replaces each feed's
// last trade with its order book mid when the book is usable (see midPrice).
// Feeds without a usable book (no book data, e.g. DEX/FX or providers that
// don't stream one; stale book; one-sided or crossed book; spread above
// MidMaxSpreadRatio) keep their last trade and its freshness check.
//
// Book-only feeds (no trade yet, see FeedData.BookOnly) have no last trade
// to fall back on: they are used only through a usable mid and are dropped
// otherwise, and always for other configs.
//
// The source feed lives in latestFeedDataMap and is shared with other
// aggregators, so a copy is emitted.
func (c *LocalAggregator) selectMidPrices(feeds []*FeedData) []*FeedData {
	useMid := c.Config.PriceSource == PriceSourceMid
	now := time.Now()
	out := make([]*FeedData, 0, len(feeds))
	for _, fd := range feeds {
		mid, ok := 0.0, false
		if useMid {
			mid, ok = midPrice(fd, now)
		}
		if !ok {
			if !fd.BookOnly {
				out = append(out, fd)
			}
			continue
		}
		fdCopy := *fd
		fdCopy.Value = mid
		fdCopy.Timestamp = fd.BookTimestamp
		fdCopy.FromBook = true
		out = append(out, &fdCopy)
	}
	return out
}

// midPrice returns the mid of the feed's best bid/ask if both sides are
// present and not crossed, the book is at most MidBookMaxAge old and the
// spread is at most MidMaxSpreadRatio of the mid.
func midPrice(fd *FeedData, now time.Time) (float64, bool) {
	if fd.Bid <= 0 || fd.Ask <= 0 || fd.Ask < fd.Bid || fd.BookTimestamp == nil {
		return 0, false
	}
	if now.Sub(*fd.BookTimestamp) > MidBookMaxAge {
		return 0, false
	}
	mid := (fd.Bid + fd.Ask) / 2
	if (fd.Ask-fd.Bid)/mid > MidMaxSpreadRatio {
		return 0, false
	}
	return mid, true
}

// applyPerFeedMultipliers walks each FeedData against its source Feed
// definition and, if the definition declares MultiplyBy, scales the value by
// the named config's most recent raw aggregate (or by 1/value when
// MultiplyByReciprocal is true).
//
// The source feed lives in latestFeedDataMap and is shared with other
// aggregators, so we never mutate it in place — we emit a copy with the
// adjusted value. Feeds whose multiplier source is missing or stale are
// dropped from this aggregation cycle so the synthetic value isn't computed
// from nonsense; non-synthetic feeds pass through unchanged.
func (c *LocalAggregator) applyPerFeedMultipliers(feeds []*FeedData) []*FeedData {
	if c.localAggregateValueMap == nil || len(c.Feeds) == 0 {
		return feeds
	}

	feedById := make(map[int32]*Feed, len(c.Feeds))
	for i := range c.Feeds {
		feedById[c.Feeds[i].ID] = &c.Feeds[i]
	}

	freshness := time.Minute
	if c.Config.FeedDataFreshness != nil && *c.Config.FeedDataFreshness > 0 {
		freshness = time.Duration(*c.Config.FeedDataFreshness) * time.Millisecond
	}

	out := make([]*FeedData, 0, len(feeds))
	for _, fd := range feeds {
		feed, ok := feedById[fd.FeedID]
		if !ok {
			out = append(out, fd)
			continue
		}
		defn := new(Definition)
		if err := json.Unmarshal(feed.Definition, defn); err != nil {
			out = append(out, fd)
			continue
		}
		if defn.MultiplyBy == nil || *defn.MultiplyBy == "" {
			out = append(out, fd)
			continue
		}
		entry, ok := c.localAggregateValueMap.Get(*defn.MultiplyBy)
		if !ok || entry.Value == 0 {
			log.Debug().Str("Player", "LocalAggregator").
				Str("config", c.Name).
				Str("feed", feed.Name).
				Str("multiplyBy", *defn.MultiplyBy).
				Msg("per-feed multiplier source not yet available, dropping feed")
			continue
		}
		if time.Since(entry.Timestamp) > freshness {
			log.Warn().Str("Player", "LocalAggregator").
				Str("config", c.Name).
				Str("feed", feed.Name).
				Str("multiplyBy", *defn.MultiplyBy).
				Dur("age", time.Since(entry.Timestamp)).
				Dur("freshness", freshness).
				Msg("per-feed multiplier source is stale, dropping feed")
			continue
		}
		newValue := fd.Value
		if defn.MultiplyByReciprocal != nil && *defn.MultiplyByReciprocal {
			newValue /= entry.Value
		} else {
			newValue *= entry.Value
		}
		fdCopy := *fd
		fdCopy.Value = newValue
		out = append(out, &fdCopy)
	}
	return out
}

func (c *LocalAggregator) filterStaleFeeds(feeds []*FeedData) []*FeedData {
	if c.Config.FeedDataFreshness == nil || *c.Config.FeedDataFreshness <= 0 {
		return feeds
	}

	if len(feeds) <= 1 {
		return feeds
	}

	// early returns above skip the age check, so they don't touch stale marks
	stalePass := c.beginStaleFeedPass(feeds)
	defer stalePass.end()

	freshness := time.Duration(*c.Config.FeedDataFreshness) * time.Millisecond
	now := time.Now()
	fresh := make([]*FeedData, 0, len(feeds))

	for _, feed := range feeds {
		// Event-driven feeds without volume (DEX swaps) can be quiet for long
		// and are not subject to freshness filtering. Polled HTTP feeds are:
		// their timestamp only advances on a successful fetch, so a failing
		// source would otherwise keep voting with its last value.
		if feed.Volume == 0 && !feed.Polled {
			fresh = append(fresh, feed)
			continue
		}

		if feed.Timestamp == nil {
			fresh = append(fresh, feed)
			continue
		}

		// A book mid is a current price while the book is unchanged, so it
		// gets the book age limit (selectMidPrices already enforced it).
		limit := freshness
		if feed.FromBook {
			limit = max(freshness, MidBookMaxAge)
		}

		age := now.Sub(*feed.Timestamp)
		if age > limit {
			stalePass.noteFeedStale(feed, age, limit)
			continue
		}
		fresh = append(fresh, feed)
	}

	return fresh
}
