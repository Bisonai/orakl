//nolint:all
package fetcher

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	prevLogger, prevLevel := log.Logger, zerolog.GlobalLevel()
	t.Cleanup(func() {
		log.Logger = prevLogger
		zerolog.SetGlobalLevel(prevLevel)
	})
	buf := &bytes.Buffer{}
	log.Logger = zerolog.New(buf)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	return buf
}

func TestFilterStaleFeeds_LogsOnlyOnTransition(t *testing.T) {
	buf := captureLogs(t)
	la := newLocalAggregatorWithFreshness(intPtr(5000))

	now := time.Now()
	old := now.Add(-time.Minute)
	staleFeeds := []*FeedData{
		{FeedID: 1, Value: 100.0, Volume: 1, Timestamp: &now},
		{FeedID: 2, Value: 101.0, Volume: 1, Timestamp: &old},
	}

	for i := 0; i < 5; i++ {
		result := la.filterStaleFeeds(staleFeeds)
		assert.Len(t, result, 1)
	}
	assert.Equal(t, 1, strings.Count(buf.String(), "excluding stale feed from aggregation"))
	assert.Equal(t, 0, strings.Count(buf.String(), "no longer excluded"))

	recovered := []*FeedData{
		{FeedID: 1, Value: 100.0, Volume: 1, Timestamp: &now},
		{FeedID: 2, Value: 101.0, Volume: 1, Timestamp: &now},
	}
	for i := 0; i < 3; i++ {
		assert.Len(t, la.filterStaleFeeds(recovered), 2)
	}
	assert.Equal(t, 1, strings.Count(buf.String(), "feed no longer excluded as stale"))

	// stale again after recovery logs again
	la.filterStaleFeeds(staleFeeds)
	assert.Equal(t, 2, strings.Count(buf.String(), "excluding stale feed from aggregation"))
}

func TestFilterStaleFeeds_TransitionPerFeed(t *testing.T) {
	buf := captureLogs(t)
	la := newLocalAggregatorWithFreshness(intPtr(5000))

	now := time.Now()
	old := now.Add(-time.Minute)
	feeds := []*FeedData{
		{FeedID: 1, Value: 100.0, Volume: 1, Timestamp: &now},
		{FeedID: 2, Value: 101.0, Volume: 1, Timestamp: &old},
		{FeedID: 3, Value: 102.0, Volume: 1, Timestamp: &old},
	}
	la.filterStaleFeeds(feeds)
	la.filterStaleFeeds(feeds)
	out := buf.String()
	assert.Equal(t, 2, strings.Count(out, "excluding stale feed from aggregation"))
	assert.Contains(t, out, `"feedID":2`)
	assert.Contains(t, out, `"feedID":3`)
}

// Early-return passes (single feed) skip the age check and leave marks alone:
// a feed count flipping 2 <-> 1 must not produce a warn/info pair per tick.
func TestFilterStaleFeeds_EarlyReturnKeepsMark(t *testing.T) {
	buf := captureLogs(t)
	la := newLocalAggregatorWithFreshness(intPtr(5000))

	now := time.Now()
	old := now.Add(-time.Minute)
	stale := []*FeedData{
		{FeedID: 1, Value: 100.0, Volume: 1, Timestamp: &now},
		{FeedID: 2, Value: 101.0, Volume: 1, Timestamp: &old},
	}
	la.filterStaleFeeds(stale)
	for i := 0; i < 3; i++ {
		la.filterStaleFeeds([]*FeedData{{FeedID: 2, Value: 101.0, Volume: 1, Timestamp: &old}})
		la.filterStaleFeeds(stale)
	}
	assert.Equal(t, 0, strings.Count(buf.String(), "feed no longer excluded as stale"))
	assert.Equal(t, 1, strings.Count(buf.String(), "excluding stale feed from aggregation"))
}

func TestFilterStaleFeeds_ConcurrentPasses(t *testing.T) {
	buf := captureLogs(t)
	la := newLocalAggregatorWithFreshness(intPtr(5000))

	now := time.Now()
	old := now.Add(-time.Minute)
	feeds := []*FeedData{
		{FeedID: 1, Value: 100.0, Volume: 1, Timestamp: &now},
		{FeedID: 2, Value: 101.0, Volume: 1, Timestamp: &old},
	}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			la.filterStaleFeeds(feeds)
		}()
	}
	wg.Wait()
	assert.Equal(t, 1, strings.Count(buf.String(), "excluding stale feed from aggregation"))
	assert.Len(t, la.staleFeeds.stale, 1)
}

// A stale feed dropped before filtering (no data, stale multiplier) is not a
// recovery: no info line, and no re-warn when it comes back still stale.
func TestFilterStaleFeeds_AbsentFeedKeepsMark(t *testing.T) {
	buf := captureLogs(t)
	la := newLocalAggregatorWithFreshness(intPtr(5000))

	now := time.Now()
	old := now.Add(-time.Minute)
	stale := []*FeedData{
		{FeedID: 1, Value: 100.0, Volume: 1, Timestamp: &now},
		{FeedID: 2, Value: 101.0, Volume: 1, Timestamp: &old},
		{FeedID: 3, Value: 102.0, Volume: 1, Timestamp: &now},
	}
	la.filterStaleFeeds(stale)
	la.filterStaleFeeds([]*FeedData{stale[0], stale[2]})
	la.filterStaleFeeds(stale)
	assert.Equal(t, 0, strings.Count(buf.String(), "feed no longer excluded as stale"))
	assert.Equal(t, 1, strings.Count(buf.String(), "excluding stale feed from aggregation"))
}

// A pass that started before the last reconciled one must not replay its
// older snapshot.
func TestStaleFeedPass_SkipsOlderPass(t *testing.T) {
	buf := captureLogs(t)
	la := newLocalAggregatorWithFreshness(intPtr(5000))
	feed := &FeedData{FeedID: 2, Value: 101.0, Volume: 1}

	older := la.beginStaleFeedPass([]*FeedData{feed})
	older.noteFeedStale(feed, time.Minute, 5*time.Second)
	newer := la.beginStaleFeedPass([]*FeedData{feed})
	newer.start = older.start.Add(time.Millisecond)
	newer.end()
	older.end()

	assert.Equal(t, 0, strings.Count(buf.String(), "excluding stale feed from aggregation"))
	assert.Empty(t, la.staleFeeds.stale)
}
