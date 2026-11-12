package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/bus"
	"bisonai.com/miko/node/pkg/libp2p/helper"
	libp2psetup "bisonai.com/miko/node/pkg/libp2p/setup"
	"bisonai.com/miko/node/pkg/raft"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
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
		t.Skip("two hosts, 200 Rafts, ~10 s")
	}
	const slowStop = 3 * time.Second // aggregators slow to stop, as on mainnet

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// a host and pubsub, and a func joining one feed's Raft on them. With
	// holdTopic, a second subscription keeps pubsub from announcing that Run
	// left the topic: no per-feed unsubscribe reaches the peer, the worst case
	// of the dropped ones.
	newNode := func(holdTopic bool) (host.Host, func(string) *raft.Raft) {
		h, err := libp2psetup.NewHost(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _ = h.Close() })
		ps, err := libp2psetup.MakePubsub(ctx, h)
		require.NoError(t, err)
		return h, func(topicName string) *raft.Raft {
			topic, err := ps.Join(topicName)
			require.NoError(t, err)
			if holdTopic {
				held, err := topic.Subscribe()
				require.NoError(t, err)
				t.Cleanup(held.Cancel)
			}
			r := raft.NewRaftNode(h, ps, topic, 100, time.Second)
			r.LeaderJob = func(context.Context) error { return nil }
			r.HandleCustomMessage = func(context.Context, raft.Message) error { return nil }
			r.PeerSilenceTimeout = time.Hour // only the leave elects
			return r
		}
	}
	hA, joinA := newNode(true)
	hB, joinB := newNode(false)

	var stopping, survivor []*raft.Raft
	for i := 0; i < feeds; i++ {
		name := fmt.Sprintf("feed-%d", i)
		stopping = append(stopping, joinA(name))
		survivor = append(survivor, joinB(name))
	}
	require.NoError(t, hB.Connect(ctx, peer.AddrInfo{ID: hA.ID(), Addrs: hA.Addrs()}))

	// the stopping node leads every feed: it runs alone first
	runsCtx, stopRuns := context.WithCancel(ctx)
	var runs sync.WaitGroup
	for _, r := range stopping {
		runs.Add(1)
		go func(r *raft.Raft) {
			defer runs.Done()
			r.Run(runsCtx)
		}(r)
	}
	require.Eventually(t, func() bool { return allRoles(stopping, raft.Leader) }, 20*time.Second, 50*time.Millisecond)
	survivorCtx, stopSurvivor := context.WithCancel(ctx)
	var survivorRuns sync.WaitGroup
	for _, r := range survivor {
		survivorRuns.Add(1)
		go func(r *raft.Raft) {
			defer survivorRuns.Done()
			r.Run(survivorCtx)
		}(r)
	}
	// before the hosts close (cleanups run last-registered first)
	t.Cleanup(func() { stopSurvivor(); survivorRuns.Wait() })
	require.Eventually(t, func() bool {
		for i, r := range survivor {
			if r.GetLeader() != stopping[i].GetHostId() || r.SubscribersCount() != 1 {
				return false
			}
		}
		return true
	}, 20*time.Second, 50*time.Millisecond, "peer does not follow on every feed")
	time.Sleep(survivor[0].CooldownPeriod) // past a first-election cooldown

	reconnects := helper.New(bus.New(10), hA)
	require.NoError(t, reconnects.Run(ctx))

	start := time.Now()
	done := make(chan struct{})
	go func() {
		defer close(done)
		handOver(shutdownSteps{
			stopAggregators: stopRuns,
			closeHost:       reconnects.CloseHost,
			waitAggregators: func(ctx context.Context) error {
				stopped := make(chan struct{})
				go func() { runs.Wait(); close(stopped) }()
				select {
				case <-stopped:
				case <-ctx.Done():
					return ctx.Err()
				}
				select {
				case <-time.After(slowStop):
				case <-ctx.Done():
				}
				return nil
			},
			flushRounds: func(context.Context) error { return nil },
		})
	}()

	require.Eventually(t, func() bool { return allRoles(survivor, raft.Leader) }, handOverFailoverBound, 10*time.Millisecond,
		"peer does not lead every feed at once")
	t.Logf("peer led every feed after %s", time.Since(start))

	<-done
	assert.GreaterOrEqual(t, time.Since(start), slowStop, "hand-over did not wait for the aggregators")
	assert.True(t, allRoles(stopping, raft.Follower), "stopping node still leads a feed")
}
