package metrics

// Per-feed (per-source) latest value/timestamp/volume, served by the node
// admin server on /api/v1/metrics. Labels are the config and feed names.
//
// A collector reads LatestFeedDataMap on scrape, so the fetch path does no
// metrics work. Only feeds of the loaded set (SetFeeds, on fetcher
// init/refresh) are emitted: feeds removed on sync disappear with the next
// refresh, and loaded feeds without data yet have no series.

import (
	"sync/atomic"

	"bisonai.com/miko/node/pkg/common/types"
	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	feedValueDesc = prometheus.NewDesc(metricnames.FeedValue,
		"Latest raw value per feed", []string{metricnames.ConfigLabel, metricnames.FeedLabel}, nil)
	feedTimestampDesc = prometheus.NewDesc(metricnames.FeedTimestamp,
		"Unix timestamp of the latest value per feed", []string{metricnames.ConfigLabel, metricnames.FeedLabel}, nil)
	feedVolumeDesc = prometheus.NewDesc(metricnames.FeedVolume,
		"Latest volume per feed (only feeds reporting volume)", []string{metricnames.ConfigLabel, metricnames.FeedLabel}, nil)

	feeds = &feedCollector{}
)

func init() {
	prometheus.MustRegister(feeds)
}

// FeedLabels names a feed in metrics.
type FeedLabels struct {
	Config string
	Feed   string
}

type feedSet struct {
	data  *types.LatestFeedDataMap
	names map[int32]FeedLabels
}

type feedCollector struct {
	set atomic.Pointer[feedSet]
}

type feedSample struct {
	labels    FeedLabels
	value     float64
	volume    float64
	timestamp float64
	hasTime   bool
}

func (c *feedCollector) setFeeds(data *types.LatestFeedDataMap, names map[int32]FeedLabels) {
	if data == nil || len(names) == 0 {
		c.set.Store(nil)
		return
	}
	copied := make(map[int32]FeedLabels, len(names))
	for id, l := range names {
		copied[id] = l
	}
	c.set.Store(&feedSet{data: data, names: copied})
}

// snapshot copies the loaded feeds' latest data under the map's read lock.
func (c *feedCollector) snapshot() []feedSample {
	set := c.set.Load()
	if set == nil {
		return nil
	}
	samples := make([]feedSample, 0, len(set.names))
	set.data.Mu.RLock()
	defer set.data.Mu.RUnlock()
	for id, labels := range set.names {
		fd, ok := set.data.FeedDataMap[id]
		if !ok || fd == nil {
			continue
		}
		s := feedSample{labels: labels, value: fd.Value, volume: fd.Volume}
		if fd.Timestamp != nil {
			s.timestamp = unixSeconds(*fd.Timestamp)
			s.hasTime = true
		}
		samples = append(samples, s)
	}
	return samples
}

func (c *feedCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- feedValueDesc
	ch <- feedTimestampDesc
	ch <- feedVolumeDesc
}

func (c *feedCollector) Collect(ch chan<- prometheus.Metric) {
	for _, s := range c.snapshot() {
		ch <- prometheus.MustNewConstMetric(feedValueDesc, prometheus.GaugeValue, s.value, s.labels.Config, s.labels.Feed)
		if s.hasTime {
			ch <- prometheus.MustNewConstMetric(feedTimestampDesc, prometheus.GaugeValue, s.timestamp, s.labels.Config, s.labels.Feed)
		}
		if s.volume != 0 {
			ch <- prometheus.MustNewConstMetric(feedVolumeDesc, prometheus.GaugeValue, s.volume, s.labels.Config, s.labels.Feed)
		}
	}
}

// SetFeeds sets the loaded feeds (feed ID -> names) emitted on scrape and the
// map their latest data is read from; feeds not in names are no longer
// emitted.
func SetFeeds(data *types.LatestFeedDataMap, names map[int32]FeedLabels) {
	feeds.setFeeds(data, names)
}
