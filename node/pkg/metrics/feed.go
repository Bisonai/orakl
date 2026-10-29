package metrics

// Per-feed (per-source) latest value/timestamp/volume, served by the node
// admin server on /api/v1/metrics. Labels are the config and feed names.
//
// A collector reads LatestFeedDataMap on scrape, so the fetch path does no
// metrics work. Only feeds of the loaded set (SetFeeds, on fetcher
// init/refresh) are emitted: feeds removed on sync disappear with the next
// refresh. A loaded feed without data (none yet, or evicted by CleanupJob)
// has only a timestamp, the time it was loaded, so its delay keeps growing.

import (
	"sync"
	"sync/atomic"
	"time"

	"bisonai.com/miko/node/pkg/common/types"
	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	feedValueDesc = prometheus.NewDesc(metricnames.FeedValue,
		"Latest raw value per feed", []string{metricnames.ConfigLabel, metricnames.FeedLabel}, nil)
	feedTimestampDesc = prometheus.NewDesc(metricnames.FeedTimestamp,
		"Unix timestamp of the latest value per feed (time the feed was loaded if none)", []string{metricnames.ConfigLabel, metricnames.FeedLabel}, nil)
	feedVolumeDesc = prometheus.NewDesc(metricnames.FeedVolume,
		"Latest volume per feed (0 if the source reports none)", []string{metricnames.ConfigLabel, metricnames.FeedLabel}, nil)

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

type loadedFeed struct {
	FeedLabels
	loadedAt time.Time
}

type feedSet struct {
	data  *types.LatestFeedDataMap
	feeds map[int32]loadedFeed
}

type feedCollector struct {
	setMu sync.Mutex // serializes setFeeds; scrapes only Load
	set   atomic.Pointer[feedSet]
}

type feedSample struct {
	labels    FeedLabels
	value     float64
	volume    float64
	timestamp float64
	hasData   bool
}

// setFeeds swaps in the loaded set; feeds already loaded keep their load time.
func (c *feedCollector) setFeeds(data *types.LatestFeedDataMap, names map[int32]FeedLabels, now time.Time) {
	c.setMu.Lock()
	defer c.setMu.Unlock()
	if data == nil || len(names) == 0 {
		c.set.Store(nil)
		return
	}
	prev := c.set.Load()
	loaded := make(map[int32]loadedFeed, len(names))
	for id, labels := range names {
		feed := loadedFeed{FeedLabels: labels, loadedAt: now}
		if prev != nil {
			if p, ok := prev.feeds[id]; ok && p.FeedLabels == labels {
				feed.loadedAt = p.loadedAt
			}
		}
		loaded[id] = feed
	}
	c.set.Store(&feedSet{data: data, feeds: loaded})
}

// snapshot copies the loaded feeds' latest data under the map's read lock.
func (c *feedCollector) snapshot() []feedSample {
	set := c.set.Load()
	if set == nil {
		return nil
	}
	samples := make([]feedSample, 0, len(set.feeds))
	set.data.Mu.RLock()
	defer set.data.Mu.RUnlock()
	for id, feed := range set.feeds {
		s := feedSample{labels: feed.FeedLabels, timestamp: unixSeconds(feed.loadedAt)}
		if fd, ok := set.data.FeedDataMap[id]; ok && fd != nil {
			s.value, s.volume, s.hasData = fd.Value, fd.Volume, true
			if fd.Timestamp != nil {
				s.timestamp = unixSeconds(*fd.Timestamp)
			}
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
		emitFeedMetric(ch, feedTimestampDesc, s.timestamp, s.labels)
		if s.hasData {
			emitFeedMetric(ch, feedValueDesc, s.value, s.labels)
			emitFeedMetric(ch, feedVolumeDesc, s.volume, s.labels)
		}
	}
}

// emitFeedMetric skips series with invalid label values (names come from the
// remote config bundle); a panic in Collect would crash the node.
func emitFeedMetric(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, labels FeedLabels) {
	m, err := prometheus.NewConstMetric(desc, prometheus.GaugeValue, value, labels.Config, labels.Feed)
	if err != nil {
		return
	}
	ch <- m
}

// SetFeeds sets the loaded feeds (feed ID -> names) emitted on scrape and the
// map their latest data is read from; feeds not in names are no longer
// emitted.
func SetFeeds(data *types.LatestFeedDataMap, names map[int32]FeedLabels) {
	feeds.setFeeds(data, names, time.Now())
}
