package cluster_test

import (
	"os"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/raft"
	"bisonai.com/miko/node/pkg/testutil/cluster"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// settleTimeout bounds the first elections of a fresh cluster.
	settleTimeout = 15 * time.Second
	// failoverBound is how soon after a leader's stop every feed has a new
	// leader: the leave grace (~0.6-0.85 s) and an election, with room for a
	// loaded runner.
	failoverBound = 3 * time.Second
	// median of the default values 1000, 1001, 1002 of three nodes
	threeNodeMedian = 1001
)

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
	os.Exit(m.Run())
}

// ledAfter returns the submissions of feed led after since.
func ledAfter(c *cluster.Cluster, feed string, since time.Time) []cluster.Submission {
	var led []cluster.Submission
	for _, s := range c.Led(feed) {
		if s.At.After(since) {
			led = append(led, s)
		}
	}
	return led
}

// waitLed waits until every feed has at least n submissions led after since.
func waitLed(t *testing.T, c *cluster.Cluster, since time.Time, n int, timeout time.Duration) {
	t.Helper()
	require.Eventually(t, func() bool {
		for _, feed := range c.Feeds() {
			if len(ledAfter(c, feed, since)) < n {
				return false
			}
		}
		return true
	}, timeout, 10*time.Millisecond, "feeds without %d led submissions:\n%s", n, c.Describe())
}

// assertRoundsNotReused checks that no round ID of a feed was used for two
// rounds. Two nodes can record the same round as led across a leader change
// (Submission.Leader); a reuse is a second round, with its own trigger
// timestamp, under the same ID.
func assertRoundsNotReused(t *testing.T, c *cluster.Cluster) {
	t.Helper()
	for _, feed := range c.Feeds() {
		seen := map[int32]cluster.Submission{}
		for _, s := range c.Led(feed) {
			if prev, ok := seen[s.Round]; ok && !prev.Timestamp.Equal(s.Timestamp) {
				t.Errorf("%s: round %d used twice (node %d at %s, node %d at %s)", feed, s.Round, prev.Node, prev.Timestamp, s.Node, s.Timestamp)
			}
			seen[s.Round] = s
		}
	}
}

// waitStable waits for a single leader per feed and no election for a
// cooldown, so a later election isn't held off by one.
func waitStable(c *cluster.Cluster) map[string]*cluster.Node {
	return c.WaitStable(c.Node(0).Raft(c.Feeds()[0]).CooldownPeriod, 2*settleTimeout)
}

// Every feed has one leader, the only node whose rounds are submitted, with
// the median of every node's value; every node gets every round.
func TestSteadyStateSingleLeaderPerFeed(t *testing.T) {
	t.Parallel()
	c := cluster.Start(t, cluster.Options{Nodes: 3})
	leaders := waitStable(c)
	settled := time.Now()

	waitLed(t, c, settled, 3, 10*time.Second)
	for _, feed := range c.Feeds() {
		for _, s := range ledAfter(c, feed, settled) {
			assert.Equal(t, leaders[feed].Index(), s.Node, "%s: round %d led by another node", feed, s.Round)
			assert.EqualValues(t, threeNodeMedian, s.Value, "%s: round %d", feed, s.Round)
		}
	}
	// every node records each round, followers with leader=false
	for _, feed := range c.Feeds() {
		round := ledAfter(c, feed, settled)[0].Round
		for _, n := range c.Nodes() {
			assert.Eventually(t, func() bool {
				for _, s := range n.Submissions() {
					if s.Feed == feed && s.Round == round {
						return s.Leader == (n == leaders[feed])
					}
				}
				return false
			}, 5*time.Second, 10*time.Millisecond, "node %d: %s round %d", n.Index(), feed, round)
		}
	}
	assertRoundsNotReused(t, c)
}

// A stopped leader's feeds get a new leader within failoverBound, which goes
// on with new rounds.
func TestLeaderStopFailsOver(t *testing.T) {
	t.Parallel()
	c := cluster.Start(t, cluster.Options{Nodes: 3})
	leaders := waitStable(c)
	stopped := leaders[c.Feeds()[0]]

	took := c.Failover(stopped.Index(), failoverBound)
	t.Logf("every feed had a single leader %s after node %d stopped", took, stopped.Index())
	failedOver := time.Now()

	for _, n := range c.Running() {
		for i, r := range n.Rafts() {
			assert.NotEqual(t, stopped.ID().String(), r.GetLeader(), "node %d still follows the stopped node on %s", n.Index(), c.Feeds()[i])
		}
	}
	for _, r := range stopped.Rafts() {
		assert.NotEqual(t, raft.Leader, r.GetRole(), "stopped node still leads")
	}
	waitLed(t, c, failedOver, 2, 10*time.Second)
	for _, feed := range c.Feeds() {
		for _, s := range ledAfter(c, feed, failedOver) {
			assert.NotEqual(t, stopped.Index(), s.Node, "%s: round %d led by the stopped node", feed, s.Round)
		}
	}
	assertRoundsNotReused(t, c)
}

// A restarted node (new peer ID, as in production) rejoins every feed's
// topic, and every feed again has one leader that all nodes follow.
func TestRestartWithNewPeerIDRejoins(t *testing.T) {
	t.Parallel()
	c := cluster.Start(t, cluster.Options{Nodes: 3})
	leaders := waitStable(c)
	n := leaders[c.Feeds()[0]] // restart a leader: failover, then rejoin
	oldID := n.ID()

	n.Restart()
	require.NotEqual(t, oldID, n.ID(), "restart kept the peer ID")
	restarted := time.Now()

	c.WaitMesh(10 * time.Second)
	c.WaitSingleLeader(10 * time.Second)
	waitLed(t, c, restarted, 2, 10*time.Second)
	require.Eventually(t, func() bool {
		for _, s := range n.Submissions() {
			if s.At.After(restarted) {
				return true
			}
		}
		return false
	}, 10*time.Second, 10*time.Millisecond, "restarted node gets no rounds")
	// peers have the new ID on every topic, and none follows the old one
	for _, m := range c.Running() {
		if m == n {
			continue
		}
		for i, r := range m.Rafts() {
			assert.Contains(t, m.Pubsub().ListPeers(r.Topic.String()), n.ID(), "node %d: %s", m.Index(), c.Feeds()[i])
			assert.NotEqual(t, oldID.String(), r.GetLeader(), "node %d follows the old ID on %s", m.Index(), c.Feeds()[i])
		}
	}
	assertRoundsNotReused(t, c)
}

// Two partitioned nodes each lead their own copy of every feed; after the
// heal they agree on one leader per feed again.
func TestPartitionAndHeal(t *testing.T) {
	t.Parallel()
	c := cluster.Start(t, cluster.Options{Nodes: 2, Feeds: []string{"feed-0", "feed-1"}})
	waitStable(c)

	c.Partition(0, 1)
	require.Eventually(t, func() bool {
		for _, n := range c.Nodes() {
			for _, r := range n.Rafts() {
				if r.GetRole() != raft.Leader {
					return false
				}
			}
		}
		return true
	}, 10*time.Second, 10*time.Millisecond, "partitioned nodes don't both lead:\n%s", c.Describe())
	// the link stays cut: no reconnect
	time.Sleep(time.Second)
	assert.Empty(t, c.Node(0).Host().Network().ConnsToPeer(c.Node(1).ID()))

	c.Heal(0, 1)
	c.WaitMesh(10 * time.Second)
	c.WaitSingleLeader(10 * time.Second)
}

// A clock offset shifts the timestamp of the rounds that node leads only.
func TestClockOffset(t *testing.T) {
	t.Parallel()
	const offset = time.Hour
	c := cluster.Start(t, cluster.Options{Nodes: 2, Feeds: []string{"feed-0", "feed-1", "feed-2", "feed-3"}})
	leaders := waitStable(c)
	c.Node(0).SetClockOffset(offset)
	settled := time.Now()
	waitLed(t, c, settled, 2, 10*time.Second)
	for feed, l := range leaders {
		want := time.Duration(0)
		if l.Index() == 0 {
			want = offset
		}
		for _, s := range ledAfter(c, feed, settled)[1:] { // [0] may predate the offset
			assert.WithinDuration(t, s.At.Add(want), s.Timestamp, 5*time.Second, "%s led by node %d", feed, l.Index())
		}
	}
}

// With the boot API, nodes find each other and a restarted node rejoins as in
// production.
func TestBootAPIDiscovery(t *testing.T) {
	c := cluster.Start(t, cluster.Options{Nodes: 2, Feeds: []string{"feed-0"}, BootAPI: true})
	c.WaitMesh(10 * time.Second)
	c.WaitSingleLeader(settleTimeout)

	c.Node(1).Restart()
	c.WaitMesh(10 * time.Second)
	c.WaitSingleLeader(10 * time.Second)
}

// A node whose dials fail at startup is connected once they work again,
// without any disconnect event (#2915).
func TestStartupDialFailureRecovers(t *testing.T) {
	c := cluster.New(t, cluster.Options{Nodes: 2, Feeds: []string{"feed-0"}, BootAPI: true})
	c.Node(0).Start()
	c.Node(1).FailDials(true)
	c.Node(1).Start() // its startup sync can't reach node 0
	c.Node(1).FailDials(false)
	c.WaitMesh(60 * time.Second) // within the reconnect backoff
	c.WaitSingleLeader(10 * time.Second)
}
