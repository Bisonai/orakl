package cluster_test

import (
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/testutil/cluster"
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

	// past an aggregate the node may still vote: max age, plus a round
	settle := maxAge + 2*c.Options().AggregateInterval
	waitValue := func(since time.Time, want int64) {
		t.Helper()
		require.Eventually(t, func() bool {
			led := ledAfter(c, feed, since)
			return len(led) > 0 && led[len(led)-1].Value == want
		}, settleTimeout, 10*time.Millisecond, "no led submission of %d:\n%s", want, c.Describe())
	}
	waitValue(time.Now(), 2000) // median of 1000 and 3000

	c.Node(1).StopValue(feed)
	time.Sleep(settle)
	since := time.Now()
	waitLed(t, c, since, 3, settleTimeout)
	for _, s := range ledAfter(c, feed, since) {
		assert.Equal(t, int64(1000), s.Value, "round %d voted node 1's frozen value", s.Round)
	}

	c.Node(0).StopValue(feed)
	time.Sleep(settle)
	since = time.Now()
	time.Sleep(5 * c.Options().AggregateInterval)
	assert.Zero(t, len(ledAfter(c, feed, since)), "frozen values submitted with no fresh local aggregate")

	c.Node(1).SetValue(feed, 3000)
	waitValue(time.Now(), 3000)
}
