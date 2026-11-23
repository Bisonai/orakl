package fetcher

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// staleFeedLog tracks which feeds of one LocalAggregator are currently
// excluded as stale, so the exclusion is logged once on the transition
// instead of on every aggregation pass. It only holds currently-stale feed
// IDs, which are a subset of the aggregator's fixed feed set (aggregators are
// recreated on refresh), so it cannot grow without bound.
type staleFeedLog struct {
	mu    sync.Mutex
	stale map[int32]struct{}
}

// markStale records feedID as stale and reports whether it was fresh before.
func (s *staleFeedLog) markStale(feedID int32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.stale[feedID]; ok {
		return false
	}
	if s.stale == nil {
		s.stale = make(map[int32]struct{})
	}
	s.stale[feedID] = struct{}{}
	return true
}

// markFresh clears feedID and reports whether it was stale before.
func (s *staleFeedLog) markFresh(feedID int32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.stale[feedID]; !ok {
		return false
	}
	delete(s.stale, feedID)
	return true
}

// noteFeedStale logs a feed being excluded as stale, only when it turns stale.
func (c *LocalAggregator) noteFeedStale(feed *FeedData, age, freshness time.Duration) {
	if !c.staleFeeds.markStale(feed.FeedID) {
		return
	}
	log.Warn().Str("Player", "LocalAggregator").
		Str("config", c.Name).
		Int32("feedID", feed.FeedID).
		Dur("age", age).
		Dur("freshness", freshness).
		Float64("value", feed.Value).
		Msg("excluding stale feed from aggregation")
}

// noteFeedFresh logs a previously stale feed being included again.
func (c *LocalAggregator) noteFeedFresh(feed *FeedData) {
	if !c.staleFeeds.markFresh(feed.FeedID) {
		return
	}
	log.Info().Str("Player", "LocalAggregator").
		Str("config", c.Name).
		Int32("feedID", feed.FeedID).
		Msg("stale feed fresh again, including in aggregation")
}
