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
	assert.Equal(t, 0, strings.Count(buf.String(), "fresh again"))

	recovered := []*FeedData{
		{FeedID: 1, Value: 100.0, Volume: 1, Timestamp: &now},
		{FeedID: 2, Value: 101.0, Volume: 1, Timestamp: &now},
	}
	for i := 0; i < 3; i++ {
		assert.Len(t, la.filterStaleFeeds(recovered), 2)
	}
	assert.Equal(t, 1, strings.Count(buf.String(), "stale feed fresh again"))

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

func TestStaleFeedLog_Concurrent(t *testing.T) {
	var s staleFeedLog
	var wg sync.WaitGroup
	var mu sync.Mutex
	transitions := 0
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if s.markStale(7) {
				mu.Lock()
				transitions++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, 1, transitions)
	assert.True(t, s.markFresh(7))
	assert.False(t, s.markFresh(7))
	assert.Empty(t, s.stale)
}
