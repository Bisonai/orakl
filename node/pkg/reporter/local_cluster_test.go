package reporter

import (
	"context"
	"sync"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/aggregator"
	"bisonai.com/orakl/node/pkg/raft"
	"bisonai.com/orakl/node/pkg/testutil/cluster"
	"github.com/stretchr/testify/require"
)

// A leader that resigns or stops a feed stops submitting it at once, though
// its last round's data is fresh and no later round reaches its reporter
// (#2938). In-node reporters are wired to the cluster's aggregators as in
// cmd/node; only their pair selection runs, nothing is sent on-chain.
func TestLocalStopsSubmittingLostFeed(t *testing.T) {
	const feed = "A-B"
	var mu sync.Mutex
	reporters := map[int]*App{}
	c := cluster.Start(t, cluster.Options{Nodes: 3, Feeds: []string{feed},
		ConfigureApp: func(n *cluster.Node, app *aggregator.App) {
			r := NewNodeApp()
			app.OnSubmission = func(data aggregator.SubmissionData, leader bool) {
				r.Ingest(data.Symbol, data.GlobalAggregate.Value, data.GlobalAggregate.Timestamp, data.Proof.Proof, leader)
			}
			r.SetLeads(app.Leads)
			mu.Lock()
			reporters[n.Index()] = r
			mu.Unlock()
		}})
	// led reports whether n's reporter would submit feed now (proof ordering
	// aside: no whitelist is set)
	led := func(n *cluster.Node) bool {
		mu.Lock()
		r := reporters[n.Index()]
		mu.Unlock()
		_, count := r.local.prepareLed(context.Background(), r.local.latest(r.LatestDataMap, []string{feed}))
		return count > 0
	}
	waitLeads := func() *cluster.Node {
		var l *cluster.Node
		require.Eventually(t, func() bool {
			l = c.Leaders()[feed]
			return l != nil && led(l)
		}, 20*time.Second, 10*time.Millisecond, "no leader submitting:\n%s", c.Describe())
		return l
	}

	// resign (unhealthy reporter): nothing submitted once it no longer leads
	sick := waitLeads()
	sick.SetCanLead(false)
	require.Eventually(t, func() bool { return sick.Role(feed) != raft.Leader }, 10*time.Second, 10*time.Millisecond)
	require.False(t, led(sick), "node %d submits a feed it resigned", sick.Index())

	// stop (/aggregator/stop, shutdown): no round follows on this node at all
	stopped := waitLeads()
	require.NotEqual(t, sick, stopped)
	stopped.StopAggregators()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, stopped.WaitAggregators(ctx))
	mu.Lock()
	r := reporters[stopped.Index()]
	mu.Unlock()
	flag, _ := r.local.leaders.Load(feed)
	require.Equal(t, true, flag, "last ingested round was led")
	require.False(t, led(stopped), "node %d submits a feed after stopping", stopped.Index())
}
