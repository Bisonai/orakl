package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/aggregator"
	"bisonai.com/miko/node/pkg/bus"
	"bisonai.com/miko/node/pkg/libp2p/helper"
	"bisonai.com/miko/node/pkg/raft"
	"bisonai.com/miko/node/pkg/testutil/cluster"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// handOverFailoverBound is how soon after the leader's SIGTERM the peer must
// lead every feed: the leave grace (~0.6-0.85 s) and a lone election, with
// room for a loaded runner; still under slowStop, which the old order waited.
const handOverFailoverBound = 2 * time.Second

// allRoles reports whether every node has the role.
func allRoles(nodes []*raft.Raft, role raft.RoleType) bool {
	for _, n := range nodes {
		if n.GetRole() != role {
			return false
		}
	}
	return true
}

// Two nodes share many feeds, all led by the stopping node. Its hand-over makes
// the peer lead every feed within about a second, even though its aggregators
// take seconds to stop and no unsubscribe reaches the peer (#2817): the host
// closes before that wait, and the disconnect drops the node from every topic
// at once. Before, the peer waited for the close: the per-feed unsubscribes,
// sent together, overflow pubsub's per-peer queue and are retried late.
func TestHandOverPeerLeadsEveryFeedAtOnce(t *testing.T) {
	const feeds = 100
	if testing.Short() {
		t.Skip("two nodes, 200 Rafts, ~10 s")
	}
	const slowStop = 3 * time.Second // aggregators slow to stop, as on mainnet

	names := make([]string, feeds)
	for i := range names {
		names[i] = fmt.Sprintf("feed-%d", i)
	}
	c := cluster.New(t, cluster.Options{
		Nodes:             2,
		Feeds:             names,
		AggregateInterval: time.Second,
		ConfigureAggregator: func(n *cluster.Node, a *aggregator.Aggregator) {
			// no rounds: leadership only (heartbeats, votes), as before the
			// harness, so the bound measures the hand-over alone
			a.Raft.LeaderJob = func(context.Context) error { return nil }
			a.Raft.HandleCustomMessage = func(context.Context, raft.Message) error { return nil }
			a.Raft.PeerSilenceTimeout = time.Hour // only the leave elects
			if n.Index() == 0 {
				// a second subscription keeps pubsub from announcing that
				// Run left the topic: no per-feed unsubscribe reaches the
				// peer, the worst case of the dropped ones
				held, err := a.Raft.Topic.Subscribe()
				require.NoError(t, err)
				t.Cleanup(held.Cancel)
			}
		},
	})
	stoppingNode := c.Node(0)

	// the stopping node leads every feed: it runs alone first
	stoppingNode.Start()
	stopping := stoppingNode.Rafts()
	require.Eventually(t, func() bool { return allRoles(stopping, raft.Leader) }, 20*time.Second, 50*time.Millisecond)
	c.Node(1).Start()
	survivor := c.Node(1).Rafts()
	require.Eventually(t, func() bool {
		for i, r := range survivor {
			if r.GetLeader() != stopping[i].GetHostId() || r.SubscribersCount() != 1 {
				return false
			}
		}
		return true
	}, 20*time.Second, 50*time.Millisecond, "peer does not follow on every feed")
	time.Sleep(survivor[0].CooldownPeriod) // past a first-election cooldown

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reconnects := helper.New(bus.New(10), stoppingNode.Host())
	require.NoError(t, reconnects.Run(ctx))

	start := time.Now()
	done := make(chan struct{})
	t.Cleanup(func() { <-done }) // a failed assertion still waits, before the nodes stop
	go func() {
		defer close(done)
		handOver(shutdownSteps{
			stopAggregators: stoppingNode.StopAggregators,
			closeHost:       reconnects.CloseHost,
			waitAggregators: func(ctx context.Context) error {
				if err := stoppingNode.WaitAggregators(ctx); err != nil {
					return err
				}
				select {
				case <-time.After(slowStop):
				case <-ctx.Done():
				}
				return nil
			},
			flushRounds: stoppingNode.FlushRounds,
		})
	}()

	require.Eventually(t, func() bool { return allRoles(survivor, raft.Leader) }, handOverFailoverBound, 10*time.Millisecond,
		"peer does not lead every feed at once")
	t.Logf("peer led every feed after %s", time.Since(start))

	<-done
	assert.GreaterOrEqual(t, time.Since(start), slowStop, "hand-over did not wait for the aggregators")
	assert.True(t, allRoles(stopping, raft.Follower), "stopping node still leads a feed")
}
