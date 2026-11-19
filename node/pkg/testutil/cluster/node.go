package cluster

import (
	"context"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/aggregator"
	"bisonai.com/miko/node/pkg/bus"
	chainhelper "bisonai.com/miko/node/pkg/chain/helper"
	"bisonai.com/miko/node/pkg/libp2p/helper"
	libp2psetup "bisonai.com/miko/node/pkg/libp2p/setup"
	"bisonai.com/miko/node/pkg/nodeconfig"
	"bisonai.com/miko/node/pkg/nodestate"
	"bisonai.com/miko/node/pkg/raft"
	"github.com/kaiachain/kaia/crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// stepTimeout bounds each wait of Node.Stop, like the production hand-over.
const stepTimeout = 5 * time.Second

// Node is one node of a cluster. Its rounds (with PersistRounds), signer,
// values and controls persist across restarts; host, pubsub and aggregators
// are new on every start.
type Node struct {
	c     *Cluster
	t     testing.TB
	index int

	state  *nodestate.Store // nil without PersistRounds
	signer *chainhelper.Signer

	failDials   atomic.Bool
	cantLead    atomic.Bool  // inverted, so the zero value is healthy
	clockOffset atomic.Int64 // time.Duration

	mu     sync.Mutex
	values map[string]int64
	run    *nodeRun // the latest start; nil before the first
	subs   []Submission
}

// nodeRun is one start of a node.
type nodeRun struct {
	host    host.Host
	ps      *pubsub.PubSub
	app     *aggregator.App
	helper  *helper.App // with BootAPI only
	rafts   map[string]*raft.Raft
	configs map[string]int32 // feed -> config ID

	cancel   context.CancelFunc // host, pubsub, helper
	stopAggs context.CancelFunc

	aggsStopped atomic.Bool
	stopOnce    sync.Once
}

func newNode(t testing.TB, c *Cluster, i int) *Node {
	t.Helper()
	var state *nodestate.Store
	if c.opts.PersistRounds {
		var err error
		state, err = nodestate.Open(c.ctx, filepath.Join(c.dir, fmt.Sprintf("node-%d.db", i)))
		if err != nil {
			t.Fatalf("cluster: node %d state: %v", i, err)
		}
	}
	pk, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("cluster: node %d key: %v", i, err)
	}
	signer, err := chainhelper.NewSigner(c.ctx, chainhelper.WithSignerPk(hex.EncodeToString(crypto.FromECDSA(pk))))
	if err != nil {
		t.Fatalf("cluster: node %d signer: %v", i, err)
	}
	n := &Node{c: c, t: t, index: i, state: state, signer: signer, values: map[string]int64{}}
	for _, feed := range c.opts.Feeds {
		n.values[feed] = c.opts.Value(i, feed)
	}
	return n
}

func (n *Node) Index() int { return n.index }

func (n *Node) current() *nodeRun {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.run
}

// mustRun returns the latest start, failing the test before the first.
func (n *Node) mustRun() *nodeRun {
	r := n.current()
	if r == nil {
		n.t.Fatalf("cluster: node %d never started", n.index)
	}
	return r
}

// ID is the peer ID of the latest start.
func (n *Node) ID() peer.ID { return n.mustRun().host.ID() }

// Host, Pubsub and App are those of the latest start.
func (n *Node) Host() host.Host        { return n.mustRun().host }
func (n *Node) Pubsub() *pubsub.PubSub { return n.mustRun().ps }
func (n *Node) App() *aggregator.App   { return n.mustRun().app }
func (n *Node) Raft(feed string) *raft.Raft {
	rf, ok := n.mustRun().rafts[feed]
	if !ok {
		n.t.Fatalf("cluster: node %d has no feed %q", n.index, feed)
	}
	return rf
}

// Rafts returns the feed Rafts of the latest start, in Options.Feeds order.
func (n *Node) Rafts() []*raft.Raft {
	r := n.mustRun()
	rafts := make([]*raft.Raft, 0, len(n.c.opts.Feeds))
	for _, feed := range n.c.opts.Feeds {
		rafts = append(rafts, r.rafts[feed])
	}
	return rafts
}

func (n *Node) Role(feed string) raft.RoleType { return n.Raft(feed).GetRole() }

// Running reports whether the node's aggregators run: started, and no stop
// (or StopAggregators) since.
func (n *Node) Running() bool {
	r := n.current()
	return r != nil && !r.aggsStopped.Load()
}

// FailDials makes every dial of this node fail (on) or work again (off). It
// holds across restarts and may be set before a start.
func (n *Node) FailDials(on bool) { n.failDials.Store(on) }

// SetCanLead sets this node's health for leadership (aggregator.App.CanLead):
// false makes it resign every feed to a live peer and hold off campaigning,
// like a node whose reporter can't submit. It holds across restarts.
func (n *Node) SetCanLead(ok bool) { n.cantLead.Store(!ok) }

// SetClockOffset shifts the clock this node stamps the rounds it leads with.
func (n *Node) SetClockOffset(d time.Duration) { n.clockOffset.Store(int64(d)) }

func (n *Node) now() time.Time { return time.Now().Add(time.Duration(n.clockOffset.Load())) }

// SetValue sets this node's local aggregate of feed, now and for later starts.
// It fails the test for a feed not in Options.Feeds.
func (n *Node) SetValue(feed string, v int64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, ok := n.values[feed]; !ok {
		n.t.Fatalf("cluster: node %d has no feed %q", n.index, feed)
	}
	n.values[feed] = v
	if n.run != nil {
		n.storeValue(n.run, feed, v)
	}
}

// storeValue stores v into r; the caller holds n.mu, so a later SetValue
// can't be overwritten by an older value.
func (n *Node) storeValue(r *nodeRun, feed string, v int64) {
	id := r.configs[feed]
	r.app.LatestLocalAggregates.Store(id, &aggregator.LocalAggregate{ConfigID: id, Value: v, Timestamp: time.Now()})
}

// Submissions returns this node's submissions in recording order.
func (n *Node) Submissions() []Submission {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]Submission(nil), n.subs...)
}

// Led returns this node's submissions of feed that it led.
func (n *Node) Led(feed string) []Submission {
	var led []Submission
	for _, s := range n.Submissions() {
		if s.Leader && s.Feed == feed {
			led = append(led, s)
		}
	}
	return led
}

func (n *Node) record(id peer.ID, data aggregator.SubmissionData, leader bool) {
	s := Submission{
		Node:      n.index,
		Peer:      id,
		Feed:      data.Symbol,
		Value:     data.GlobalAggregate.Value,
		Round:     data.GlobalAggregate.Round,
		Timestamp: data.GlobalAggregate.Timestamp,
		Proof:     data.Proof.Proof,
		Leader:    leader,
		At:        time.Now(),
	}
	n.mu.Lock()
	n.subs = append(n.subs, s)
	n.mu.Unlock()
	n.c.record(s)
}

// Start starts the node on a new host, so with a new peer ID, connects it
// (see Options.BootAPI) and starts its aggregators. It fails the test if the
// node is running.
func (n *Node) Start() {
	n.t.Helper()
	if prev := n.current(); prev != nil {
		if !prev.aggsStopped.Load() {
			n.t.Fatalf("cluster: node %d already running", n.index)
		}
		n.stop() // finish a stop done step by step
	}
	if err := n.start(); err != nil {
		n.t.Fatalf("cluster: start node %d: %v", n.index, err)
	}
}

func (n *Node) start() error {
	ctx, cancel := context.WithCancel(n.c.ctx)
	r := &nodeRun{cancel: cancel, rafts: map[string]*raft.Raft{}, configs: map[string]int32{}}
	fail := func(err error) error {
		cancel()
		if r.host != nil {
			_ = r.host.Close()
		}
		return err
	}

	h, err := libp2psetup.NewHost(ctx, libp2psetup.WithConnectionGater(gater{n: n}))
	if err != nil {
		return fail(err)
	}
	r.host = h
	n.c.register(h.ID(), n.index)
	ps, err := libp2psetup.MakePubsub(ctx, h)
	if err != nil {
		return fail(err)
	}
	r.ps = ps

	// connect before the aggregators start, as nodebin does
	mb := bus.New(100)
	var bootPeers []peer.ID
	if n.c.boot != nil {
		r.helper = helper.New(mb, h)
		// a failed dial is no error here, only the boot API or the host (no
		// non-loopback address); nodebin retries that, the harness fails
		if bootPeers, err = libp2psetup.SyncThroughBootApi(ctx, h); err != nil {
			return fail(fmt.Errorf("boot sync: %w", err))
		}
	} else {
		for _, m := range n.c.Running() {
			if m == n || !n.c.linked(n.index, m.index) {
				continue
			}
			if err := n.dialFrom(h, m); err != nil {
				n.t.Logf("cluster: node %d to node %d: %v", n.index, m.index, err)
			}
		}
	}

	app := aggregator.New(mb, h, ps, n.state)
	app.Configs = nodeconfig.New()
	interval := int32(n.c.opts.AggregateInterval / time.Millisecond)
	entries := make([]nodeconfig.Entry, 0, len(n.c.opts.Feeds))
	for _, feed := range n.c.opts.Feeds {
		entries = append(entries, nodeconfig.Entry{Name: feed, AggregateInterval: &interval})
	}
	if err := app.Configs.Set(entries); err != nil {
		return fail(err)
	}
	for _, cfg := range app.Configs.Configs() {
		r.configs[cfg.Name] = cfg.ID
	}
	app.Signer = n.signer
	r.app = app
	if n.c.opts.ConfigureApp != nil {
		n.c.opts.ConfigureApp(n, app)
	}
	userCanLead := app.CanLead
	app.CanLead = func() bool {
		return !n.cantLead.Load() && (userCanLead == nil || userCanLead())
	}
	id := h.ID()
	userSubmission := app.OnSubmission
	app.OnSubmission = func(data aggregator.SubmissionData, leader bool) {
		n.record(id, data, leader)
		if userSubmission != nil {
			userSubmission(data, leader)
		}
	}
	userConfigure := app.ConfigureAggregator
	app.ConfigureAggregator = func(a *aggregator.Aggregator) {
		a.Now = n.now
		if userConfigure != nil {
			userConfigure(a)
		}
		if n.c.opts.ConfigureAggregator != nil {
			n.c.opts.ConfigureAggregator(n, a)
		}
	}
	n.storeValues(r) // before the first round

	aggCtx, stopAggs := context.WithCancel(ctx)
	r.stopAggs = stopAggs
	if err := app.Run(aggCtx); err != nil {
		stopAggs()
		// some aggregators may run already: none may outlive the test
		waitCtx, cancelWait := context.WithTimeout(context.Background(), stepTimeout)
		_ = app.WaitStopped(waitCtx)
		cancelWait()
		return fail(err)
	}
	for _, a := range app.Aggregators {
		r.rafts[a.Name] = a.Raft
	}

	if r.helper != nil {
		r.helper.SeedPending(bootPeers) // as nodebin: retry listed peers not connected
		if err := r.helper.Run(ctx); err != nil {
			stopAggs()
			// the aggregators run already: none may outlive the test
			waitCtx, cancelWait := context.WithTimeout(context.Background(), stepTimeout)
			_ = app.WaitStopped(waitCtx)
			cancelWait()
			return fail(err)
		}
	}

	n.mu.Lock()
	n.run = r
	n.mu.Unlock()
	// again: a SetValue during the start stored into the previous run
	n.storeValues(r)
	return nil
}

// storeValues stores every value of the node into r.
func (n *Node) storeValues(r *nodeRun) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for feed, v := range n.values {
		n.storeValue(r, feed, v)
	}
}

func (n *Node) dial(m *Node) error { return n.dialFrom(n.mustRun().host, m) }

func (n *Node) dialFrom(h host.Host, m *Node) error {
	target := m.current()
	if target == nil {
		return fmt.Errorf("node %d never started", m.index)
	}
	ctx, cancel := context.WithTimeout(n.c.ctx, dialTimeout)
	defer cancel()
	return h.Connect(ctx, peer.AddrInfo{ID: target.host.ID(), Addrs: target.host.Addrs()})
}

// Restart stops the node (if running) and starts it with a new peer ID.
func (n *Node) Restart() {
	n.t.Helper()
	n.Stop()
	n.Start()
}

// Stop stops the node in the production shutdown order: StopAggregators,
// CloseHost, WaitAggregators, FlushRounds. Safe to call more than once.
func (n *Node) Stop() {
	n.t.Helper()
	n.stop()
}

// stop is Stop without the test helper: it may run on any goroutine.
func (n *Node) stop() {
	r := n.current()
	if r == nil {
		return
	}
	r.stopOnce.Do(func() {
		n.StopAggregators()
		if err := n.CloseHost(); err != nil {
			n.t.Errorf("cluster: node %d close host: %v", n.index, err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), stepTimeout)
		if err := n.WaitAggregators(ctx); err != nil {
			n.t.Errorf("cluster: node %d aggregators still running: %v", n.index, err)
		}
		cancel()
		ctx, cancel = context.WithTimeout(context.Background(), stepTimeout)
		if err := n.FlushRounds(ctx); err != nil {
			n.t.Errorf("cluster: node %d flush rounds: %v", n.index, err)
		}
		cancel()
		r.cancel()
	})
}

// StopAggregators cancels every aggregator of the node (each resigns). From
// then on the node is not Running.
func (n *Node) StopAggregators() {
	r := n.mustRun()
	r.aggsStopped.Store(true)
	r.stopAggs()
}

// CloseHost closes the node's host (through the libp2p helper with BootAPI,
// as nodebin does); with BootAPI the node also leaves the boot list.
func (n *Node) CloseHost() error {
	r := n.mustRun()
	if n.c.boot != nil {
		n.c.boot.drop(r.host.ID())
		return r.helper.CloseHost()
	}
	return r.host.Close()
}

// WaitAggregators waits until every aggregator of the node has stopped.
func (n *Node) WaitAggregators(ctx context.Context) error {
	return n.mustRun().app.WaitStopped(ctx)
}

// FlushRounds persists the node's rounds.
func (n *Node) FlushRounds(ctx context.Context) error {
	return n.mustRun().app.FlushRounds(ctx)
}
