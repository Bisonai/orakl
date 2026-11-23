package fetcher

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// staleFeedLog remembers which feeds of one LocalAggregator were excluded as
// stale in the last filterStaleFeeds pass, so exclusion is logged once on the
// transition instead of on every aggregation pass. It only holds feed IDs
// from the aggregator's fixed feed set (aggregators are recreated on
// refresh), so it cannot grow without bound.
type staleFeedLog struct {
	mu    sync.Mutex
	stale map[int32]struct{}
}

type staleExclusion struct {
	feed      *FeedData
	age       time.Duration
	freshness time.Duration
}

// staleFeedPass collects the exclusions of one filterStaleFeeds call.
type staleFeedPass struct {
	aggregator *LocalAggregator
	excluded   []staleExclusion
}

func (c *LocalAggregator) beginStaleFeedPass() *staleFeedPass {
	return &staleFeedPass{aggregator: c}
}

// noteFeedStale records a feed excluded as stale in this pass.
func (p *staleFeedPass) noteFeedStale(feed *FeedData, age, freshness time.Duration) {
	p.excluded = append(p.excluded, staleExclusion{feed: feed, age: age, freshness: freshness})
}

// end reconciles this pass with the previous one: feeds newly excluded log a
// warning, feeds excluded before but not now (fresh again, or not filtered
// for any other reason) log an info line. The whole reconcile runs under the
// lock, so overlapping jobs apply whole passes in some order and never lose
// or duplicate a transition within one pass.
func (p *staleFeedPass) end() {
	c := p.aggregator
	s := &c.staleFeeds
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(p.excluded) == 0 && len(s.stale) == 0 {
		return
	}

	cur := make(map[int32]struct{}, len(p.excluded))
	for _, e := range p.excluded {
		cur[e.feed.FeedID] = struct{}{}
		if _, ok := s.stale[e.feed.FeedID]; ok {
			continue
		}
		log.Warn().Str("Player", "LocalAggregator").
			Str("config", c.Name).
			Int32("feedID", e.feed.FeedID).
			Dur("age", e.age).
			Dur("freshness", e.freshness).
			Float64("value", e.feed.Value).
			Msg("excluding stale feed from aggregation")
	}
	for feedID := range s.stale {
		if _, ok := cur[feedID]; ok {
			continue
		}
		log.Info().Str("Player", "LocalAggregator").
			Str("config", c.Name).
			Int32("feedID", feedID).
			Msg("feed no longer excluded as stale")
	}
	s.stale = cur
}
