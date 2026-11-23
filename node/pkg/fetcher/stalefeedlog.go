package fetcher

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// staleFeedLog remembers which feeds of one LocalAggregator are excluded as
// stale, so exclusion is logged once on the transition instead of on every
// aggregation pass. It only holds feed IDs from the aggregator's fixed feed
// set (aggregators are recreated on refresh), so it cannot grow without bound.
type staleFeedLog struct {
	mu    sync.Mutex
	stale map[int32]struct{}
	// start of the last reconciled pass; older passes are skipped
	lastPass time.Time
}

type staleExclusion struct {
	feed      *FeedData
	age       time.Duration
	freshness time.Duration
}

// staleFeedPass collects the exclusions of one filterStaleFeeds call.
type staleFeedPass struct {
	aggregator *LocalAggregator
	start      time.Time
	seen       []*FeedData
	excluded   []staleExclusion
}

// beginStaleFeedPass starts a pass over feeds, the input of filterStaleFeeds.
func (c *LocalAggregator) beginStaleFeedPass(feeds []*FeedData) *staleFeedPass {
	return &staleFeedPass{aggregator: c, start: time.Now(), seen: feeds}
}

// noteFeedStale records a feed excluded as stale in this pass.
func (p *staleFeedPass) noteFeedStale(feed *FeedData, age, freshness time.Duration) {
	p.excluded = append(p.excluded, staleExclusion{feed: feed, age: age, freshness: freshness})
}

// end reconciles this pass with the stored state: feeds newly excluded log a
// warning; marked feeds present in this pass but not excluded (fresh again,
// or passed through unfiltered) log an info line and are cleared. Marked
// feeds absent from this pass (dropped earlier, e.g. no data or stale
// multiplier) keep their mark. The reconcile runs under the lock, and a pass
// that started before the last reconciled one is skipped, so overlapping
// jobs cannot replay an older snapshot.
func (p *staleFeedPass) end() {
	c := p.aggregator
	s := &c.staleFeeds
	s.mu.Lock()
	defer s.mu.Unlock()

	if p.start.Before(s.lastPass) {
		return
	}
	s.lastPass = p.start

	if len(p.excluded) == 0 && len(s.stale) == 0 {
		return
	}

	excluded := make(map[int32]struct{}, len(p.excluded))
	for _, e := range p.excluded {
		excluded[e.feed.FeedID] = struct{}{}
		if _, ok := s.stale[e.feed.FeedID]; ok {
			continue
		}
		if s.stale == nil {
			s.stale = make(map[int32]struct{})
		}
		s.stale[e.feed.FeedID] = struct{}{}
		log.Warn().Str("Player", "LocalAggregator").
			Str("config", c.Name).
			Int32("feedID", e.feed.FeedID).
			Dur("age", e.age).
			Dur("freshness", e.freshness).
			Float64("value", e.feed.Value).
			Msg("excluding stale feed from aggregation")
	}
	for _, feed := range p.seen {
		if _, ok := s.stale[feed.FeedID]; !ok {
			continue
		}
		if _, ok := excluded[feed.FeedID]; ok {
			continue
		}
		delete(s.stale, feed.FeedID)
		log.Info().Str("Player", "LocalAggregator").
			Str("config", c.Name).
			Int32("feedID", feed.FeedID).
			Msg("feed no longer excluded as stale")
	}
}
