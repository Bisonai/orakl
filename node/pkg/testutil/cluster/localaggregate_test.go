package cluster_test

import (
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/testutil/cluster"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A node whose fetcher stops publishing a feed stops contributing it once its
// last local aggregate is older than the max age; when no node has a fresh
// one, rounds produce no aggregate instead of re-submitting a frozen price
// under fresh round timestamps (#2964).
func TestStaleLocalAggregateNotVoted(t *testing.T) {
	const maxAge = time.Second
	// read by aggregator.New; t.Setenv: not parallel
	t.Setenv("LOCAL_AGGREGATE_MAX_AGE", maxAge.String())
	const feed = "feed-0"
	c := cluster.Start(t, cluster.Options{
		Nodes: 2,
		Feeds: []string{feed},
		Value: func(node int, _ string) int64 { return 1000 + 2000*int64(node) },
	})
	waitStable(c)

	// rounds triggered after since: a round's prices are collected right
	// after its trigger, unlike its submission (At), which can lag
	triggeredAfter := func(since time.Time) []cluster.Submission {
		var led []cluster.Submission
		for _, s := range c.Led(feed) {
			if s.Timestamp.After(since) {
				led = append(led, s)
			}
		}
		return led
	}
	waitValue := func(since time.Time, want int64) {
		t.Helper()
		require.Eventually(t, func() bool {
			led := triggeredAfter(since)
			return len(led) > 0 && led[len(led)-1].Value == want
		}, settleTimeout, 10*time.Millisecond, "no led submission of %d:\n%s", want, c.Describe())
	}
	waitValue(time.Now(), 2000) // median of 1000 and 3000

	// past the max age, with a margin for the last store racing StopValue
	settle := func(stop time.Time) time.Time { return stop.Add(maxAge + c.Options().AggregateInterval) }

	c.Node(1).StopValue(feed)
	since := settle(time.Now())
	require.Eventually(t, func() bool { return len(triggeredAfter(since)) >= 3 }, settleTimeout, 10*time.Millisecond, "no rounds after node 1 stopped:\n%s", c.Describe())
	for _, s := range triggeredAfter(since) {
		assert.Equal(t, int64(1000), s.Value, "round %d voted node 1's frozen value", s.Round)
	}

	c.Node(0).StopValue(feed)
	since = settle(time.Now())
	time.Sleep(time.Until(since) + 5*c.Options().AggregateInterval)
	assert.Zero(t, len(triggeredAfter(since)), "frozen values submitted with no fresh local aggregate")

	c.Node(1).SetValue(feed, 3000)
	waitValue(time.Now(), 3000)
}
