package cluster

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/aggregator"
	"bisonai.com/miko/node/pkg/raft"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	defaultNodes             = 3
	defaultAggregateInterval = 500 * time.Millisecond
	pollInterval             = 10 * time.Millisecond
	dialTimeout              = 2 * time.Second
)

// Options configure a cluster; the zero value is 3 nodes and 3 feeds.
type Options struct {
	// Nodes is the number of nodes (default 3).
	Nodes int
	// Feeds are the config names every node aggregates (default feed-0..2).
	Feeds []string
	// AggregateInterval is every feed's round interval (default 500 ms).
	AggregateInterval time.Duration
	// Value is the local aggregate injected for a node and feed (default
	// 1000 + node index). Node.SetValue changes it later.
	Value func(node int, feed string) int64
	// BootAPI makes nodes discover each other through an in-process boot API
	// and reconnect through the libp2p helper, as in production (see the
	// package doc). Default: a starting node dials every running node.
	BootAPI bool
	// PersistRounds gives every node a SQLite node state (in t.TempDir), so
	// a restarted node continues after its last round, as in production.
	// Default: rounds are in memory only, and a restarted node starts at
	// round 1 (as on a fresh state). modernc SQLite trips the race detector
	// when several nodes use it at once, so tests with it don't pass -race.
	PersistRounds bool

	// ConfigureApp, if set, is called with each node's aggregator app before
	// it runs, on every start. Submissions are recorded around a hook set
	// here (OnSubmission).
	ConfigureApp func(n *Node, app *aggregator.App)
	// ConfigureAggregator, if set, is called with each aggregator before it
	// starts, on every start (e.g. raft timers).
	ConfigureAggregator func(n *Node, a *aggregator.Aggregator)
	// OnSubmission, if set, sees every recorded submission. It must not block.
	OnSubmission func(Submission)
}

// Submission is what a node's aggregator handed to the in-node reporter.
type Submission struct {
	Node      int     // index of the node
	Peer      peer.ID // the node's peer ID at the time
	Feed      string
	Value     int64
	Round     int32
	Timestamp time.Time // the round's timestamp (signed in the proof)
	Proof     []byte
	// Leader is whether the node led the round: only then does the in-node
	// reporter submit it on-chain.
	Leader bool
	At     time.Time // when it was recorded
}

// Cluster is a set of in-process nodes; see the package doc.
type Cluster struct {
	t     testing.TB
	opts  Options
	ctx   context.Context
	dir   string // node states (PersistRounds)
	nodes []*Node
	boot  *bootAPI

	mu   sync.Mutex
	ids  map[peer.ID]int // every peer ID a node ever had -> node index
	cuts map[[2]int]bool // partitioned node pairs, lower index first
	subs []Submission    // all nodes, in recording order
}

// New builds a cluster without starting any node. Everything it starts is
// stopped and closed when the test ends.
func New(t testing.TB, opts Options) *Cluster {
	t.Helper()
	if opts.Nodes == 0 {
		opts.Nodes = defaultNodes
	}
	if len(opts.Feeds) == 0 {
		opts.Feeds = []string{"feed-0", "feed-1", "feed-2"}
	}
	if opts.AggregateInterval == 0 {
		opts.AggregateInterval = defaultAggregateInterval
	}
	if opts.Value == nil {
		opts.Value = func(node int, _ string) int64 { return 1000 + int64(node) }
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := &Cluster{
		t:    t,
		opts: opts,
		ctx:  ctx,
		dir:  t.TempDir(), // before the cleanup below, so removed after it
		ids:  map[peer.ID]int{},
		cuts: map[[2]int]bool{},
	}
	if opts.BootAPI {
		c.boot = newBootAPI()
		t.Setenv("BOOT_API_URL", c.boot.server.URL)
		t.Setenv("HOST_IP", "")
	}
	// registered first, so it runs after the test's own cleanups
	t.Cleanup(func() {
		var wg sync.WaitGroup
		for _, n := range c.nodes {
			wg.Add(1)
			go func(n *Node) {
				defer wg.Done()
				n.stop()
			}(n)
		}
		wg.Wait()
		cancel()
		// node states stay open, as in nodebin: an aggregator app's round
		// persister may still flush after its stop, and closing the SQLite
		// handle under it crashes modernc
		if c.boot != nil {
			c.boot.close()
		}
	})

	for i := 0; i < opts.Nodes; i++ {
		c.nodes = append(c.nodes, newNode(t, c, i))
	}
	return c
}

// Start builds a cluster and starts every node.
func Start(t testing.TB, opts Options) *Cluster {
	t.Helper()
	c := New(t, opts)
	c.StartAll()
	return c
}

// StartAll starts every node that is not running, one after the other.
func (c *Cluster) StartAll() {
	c.t.Helper()
	for _, n := range c.nodes {
		if !n.Running() {
			n.Start()
		}
	}
}

func (c *Cluster) Node(i int) *Node { return c.nodes[i] }
func (c *Cluster) Nodes() []*Node   { return append([]*Node(nil), c.nodes...) }
func (c *Cluster) Feeds() []string  { return append([]string(nil), c.opts.Feeds...) }
func (c *Cluster) Options() Options { return c.opts }

// Running returns the nodes whose aggregators run.
func (c *Cluster) Running() []*Node {
	var running []*Node
	for _, n := range c.nodes {
		if n.Running() {
			running = append(running, n)
		}
	}
	return running
}

func pair(i, j int) [2]int {
	if i > j {
		i, j = j, i
	}
	return [2]int{i, j}
}

// Partition cuts the link between nodes i and j: their connections close and
// both refuse to connect to each other (with any peer ID) until Heal.
func (c *Cluster) Partition(i, j int) {
	c.mu.Lock()
	c.cuts[pair(i, j)] = true
	c.mu.Unlock()
	a, b := c.nodes[i].current(), c.nodes[j].current()
	if a != nil && b != nil {
		_ = a.host.Network().ClosePeer(b.host.ID())
		_ = b.host.Network().ClosePeer(a.host.ID())
	}
}

// Heal restores the link between nodes i and j. Without BootAPI it also dials
// j from i; with BootAPI reconnecting is left to the nodes.
func (c *Cluster) Heal(i, j int) {
	c.t.Helper()
	c.mu.Lock()
	delete(c.cuts, pair(i, j))
	c.mu.Unlock()
	if c.boot == nil && c.nodes[i].Running() && c.nodes[j].Running() {
		if err := c.nodes[i].dial(c.nodes[j]); err != nil {
			c.t.Logf("cluster: heal: node %d to node %d: %v", i, j, err)
		}
	}
}

// cutOff reports whether node i must not connect to peer p.
func (c *Cluster) cutOff(i int, p peer.ID) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	j, ok := c.ids[p]
	return ok && c.cuts[pair(i, j)]
}

func (c *Cluster) linked(i, j int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return !c.cuts[pair(i, j)]
}

func (c *Cluster) register(id peer.ID, i int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ids[id] = i
}

func (c *Cluster) record(s Submission) {
	c.mu.Lock()
	c.subs = append(c.subs, s)
	c.mu.Unlock()
	if c.opts.OnSubmission != nil {
		c.opts.OnSubmission(s)
	}
}

// Submissions returns every node's submissions in recording order.
func (c *Cluster) Submissions() []Submission {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Submission(nil), c.subs...)
}

// Led returns the submissions of feed whose node led the round (what the
// in-node reporters submit on-chain), in recording order.
func (c *Cluster) Led(feed string) []Submission {
	var led []Submission
	for _, s := range c.Submissions() {
		if s.Leader && s.Feed == feed {
			led = append(led, s)
		}
	}
	return led
}

// Leaders returns, per feed, the running node that leads it when exactly one
// running node does and every other running node follows it; nil otherwise.
func (c *Cluster) Leaders() map[string]*Node {
	running := c.Running()
	leaders := make(map[string]*Node, len(c.opts.Feeds))
	for _, feed := range c.opts.Feeds {
		leaders[feed] = singleLeader(running, feed)
	}
	return leaders
}

func singleLeader(running []*Node, feed string) *Node {
	var leader *Node
	for _, n := range running {
		if n.Role(feed) == raft.Leader {
			if leader != nil {
				return nil
			}
			leader = n
		}
	}
	if leader == nil {
		return nil
	}
	id := leader.ID().String()
	for _, n := range running {
		if n != leader && n.Raft(feed).GetLeader() != id {
			return nil
		}
	}
	return leader
}

// WaitSingleLeader waits until every feed has exactly one leader that every
// running node follows, and returns the leader per feed. It fails the test on
// timeout.
func (c *Cluster) WaitSingleLeader(timeout time.Duration) map[string]*Node {
	c.t.Helper()
	leaders, ok := c.waitSingleLeader(timeout)
	if !ok {
		c.t.Fatalf("cluster: no single leader per feed within %s:\n%s", timeout, c.Describe())
	}
	return leaders
}

func (c *Cluster) waitSingleLeader(timeout time.Duration) (map[string]*Node, bool) {
	deadline := time.Now().Add(timeout)
	for {
		leaders := c.Leaders()
		ok := true
		for _, l := range leaders {
			if l == nil {
				ok = false
				break
			}
		}
		if ok {
			return leaders, true
		}
		if time.Now().After(deadline) {
			return leaders, false
		}
		time.Sleep(pollInterval)
	}
}

// Failover stops node i (Node.Stop) and returns how long after the stop began
// every feed had a single leader again. It fails the test on timeout.
func (c *Cluster) Failover(i int, timeout time.Duration) time.Duration {
	c.t.Helper()
	n := c.nodes[i]
	start := time.Now()
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		n.stop()
	}()
	// the node counts as stopped from the stop's first step
	for n.Running() {
		time.Sleep(time.Millisecond)
	}
	_, ok := c.waitSingleLeader(timeout)
	took := time.Since(start)
	<-stopped
	if !ok {
		c.t.Fatalf("cluster: no single leader per feed within %s of stopping node %d:\n%s", timeout, i, c.Describe())
	}
	return took
}

// WaitStable waits until every feed has a single leader (WaitSingleLeader)
// and no running node's term of any feed changed for window, and returns the
// leaders. With window at least the raft CooldownPeriod no node is in an
// election's cooldown, which would hold off a failover. It fails the test on
// timeout.
func (c *Cluster) WaitStable(window, timeout time.Duration) map[string]*Node {
	c.t.Helper()
	deadline := time.Now().Add(timeout)
	terms := func() map[*raft.Raft]int {
		m := map[*raft.Raft]int{}
		for _, n := range c.Running() {
			for _, r := range n.Rafts() {
				m[r] = r.GetCurrentTerm()
			}
		}
		return m
	}
	for {
		leaders, ok := c.waitSingleLeader(time.Until(deadline))
		if !ok {
			c.t.Fatalf("cluster: no single leader per feed within %s:\n%s", timeout, c.Describe())
		}
		before, since := terms(), time.Now()
		stable := true
		for stable && time.Since(since) < window {
			time.Sleep(pollInterval)
			now := terms()
			for r, term := range before {
				if now[r] != term {
					stable = false
				}
			}
			if len(now) != len(before) {
				stable = false
			}
		}
		if stable {
			if again := c.Leaders(); sameLeaders(leaders, again) {
				return leaders
			}
		}
		if time.Now().After(deadline) {
			c.t.Fatalf("cluster: terms not stable for %s within %s:\n%s", window, timeout, c.Describe())
		}
	}
}

func sameLeaders(a, b map[string]*Node) bool {
	for feed, l := range a {
		if b[feed] != l {
			return false
		}
	}
	return true
}

// WaitMesh waits until every two running nodes that are not partitioned are
// connected and see each other on every feed's topic. It fails the test on
// timeout.
func (c *Cluster) WaitMesh(timeout time.Duration) {
	c.t.Helper()
	deadline := time.Now().Add(timeout)
	for !c.meshed() {
		if time.Now().After(deadline) {
			c.t.Fatalf("cluster: no full mesh within %s:\n%s", timeout, c.Describe())
		}
		time.Sleep(pollInterval)
	}
}

func (c *Cluster) meshed() bool {
	running := c.Running()
	for _, a := range running {
		ra := a.current()
		want := 0
		for _, b := range running {
			if a == b || !c.linked(a.index, b.index) {
				continue
			}
			want++
			if ra.host.Network().Connectedness(b.ID()) != network.Connected {
				return false
			}
		}
		for _, feed := range c.opts.Feeds {
			if a.Raft(feed).SubscribersCount() != want {
				return false
			}
		}
	}
	return true
}

// Describe returns every node's state per feed, for failure messages.
func (c *Cluster) Describe() string {
	var b strings.Builder
	for _, n := range c.nodes {
		r := n.current()
		if r == nil {
			fmt.Fprintf(&b, "node %d: never started\n", n.index)
			continue
		}
		fmt.Fprintf(&b, "node %d %s running=%v peers=%d\n", n.index, short(r.host.ID()), n.Running(), len(r.host.Network().Peers()))
		feeds := append([]string(nil), c.opts.Feeds...)
		sort.Strings(feeds)
		for _, feed := range feeds {
			rf := r.rafts[feed]
			leader := rf.GetLeader()
			if id, err := peer.Decode(leader); err == nil {
				leader = short(id)
			}
			fmt.Fprintf(&b, "  %s: %s term=%d leader=%s subscribers=%d\n", feed, rf.GetRole(), rf.GetCurrentTerm(), leader, rf.SubscribersCount())
		}
	}
	return b.String()
}

func short(id peer.ID) string {
	s := id.String()
	if len(s) > 8 {
		return s[len(s)-8:]
	}
	return s
}
