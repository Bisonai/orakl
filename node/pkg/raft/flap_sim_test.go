package raft

import (
	"context"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	libp2psetup "bisonai.com/miko/node/pkg/libp2p/setup"
	"bisonai.com/miko/node/pkg/utils/calculator"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/stretchr/testify/require"
)

// Two-node flap simulation. Each node has its own pubsub with no peers; a
// forwarder hands what one publishes to the other's handleMessage, so a link
// can be stalled per direction. The round protocol mirrors pkg/aggregator
// (trigger, price, price fix, proof) and counts submissions whose proofs are
// for different values, which the DAL rejects. Only uses raft API that also
// exists before v0.0.28, so it runs on both.

const (
	simTrigger MessageType = "trigger"
	simPrice   MessageType = "priceData"
	simFix     MessageType = "priceFix"
	simProof   MessageType = "proof"
)

type simMsg struct {
	Round int32     `json:"round"`
	Value int64     `json:"value"`
	TS    time.Time `json:"ts"`
}

type simLink struct {
	mu       sync.Mutex
	stallEnd time.Time
	drop     bool // a stalled link drops, else it queues
	latency  time.Duration
	queue    chan simQueued
}

type simQueued struct {
	raw  *pubsub.Message
	sent time.Time
}

func (l *simLink) stall(d time.Duration) {
	l.mu.Lock()
	l.stallEnd = time.Now().Add(d)
	l.mu.Unlock()
}

// open waits out a queueing stall, or reports false for a dropping one.
func (l *simLink) open() bool {
	for {
		l.mu.Lock()
		end, drop := l.stallEnd, l.drop
		l.mu.Unlock()
		if !time.Now().Before(end) {
			return true
		}
		if drop {
			return false
		}
		time.Sleep(time.Until(end))
	}
}

type simRound struct {
	triggered, priced, fixed, proved bool
	prices                           []int64
	priceFrom                        map[string]bool
	proofs                           []simMsg
	proofFrom                        map[string]bool
}

// simNode mirrors pkg/aggregator: one mutex per round store collapsed into one.
type simNode struct {
	r       *Raft
	value   int64
	collect time.Duration

	mu     sync.Mutex
	round  int32
	rounds map[int32]*simRound

	submissions atomic.Int64
	mixed       atomic.Int64 // proofs for different values: DAL "signer not whitelisted"
	zombie      atomic.Int64 // triggers published while not Role == Leader (stray loop)
}

func (n *simNode) get(round int32) *simRound {
	s, ok := n.rounds[round]
	if !ok {
		s = &simRound{priceFrom: map[string]bool{}, proofFrom: map[string]bool{}}
		n.rounds[round] = s
	}
	return s
}

// keepLast10 mirrors leaveOnlyLast10Entries: keeps (round-10, round], also
// dropping rounds above it.
func (n *simNode) keepLast10(round int32) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for id := range n.rounds {
		if id > round || id <= round-10 {
			delete(n.rounds, id)
		}
	}
}

func (n *simNode) publish(ctx context.Context, typ MessageType, m simMsg) error {
	data, err := encodeInner(m)
	if err != nil {
		return err
	}
	return n.r.PublishMessage(ctx, Message{Type: typ, SentFrom: n.r.GetHostId(), Data: data})
}

func (n *simNode) leaderJob(ctx context.Context) error {
	if n.r.GetRole() != Leader {
		n.zombie.Add(1)
	}
	// like Aggregator.LeaderJob: mutex held across publish, so rounds go out in order
	n.mu.Lock()
	defer n.mu.Unlock()
	n.round++
	return n.publish(ctx, simTrigger, simMsg{Round: n.round, TS: time.Now()})
}

func (n *simNode) handleCustom(ctx context.Context, msg Message) error {
	var m simMsg
	if err := decodeInner(msg.Data, &m); err != nil {
		return err
	}
	switch msg.Type {
	case simTrigger:
		defer n.keepLast10(m.Round)
		if msg.SentFrom != n.r.GetLeader() {
			return nil
		}
		n.mu.Lock()
		if msg.SentFrom != n.r.GetHostId() {
			n.round = max(n.round, m.Round)
		}
		s := n.get(m.Round)
		if s.triggered {
			n.mu.Unlock()
			return nil
		}
		s.triggered = true
		n.mu.Unlock()
		return n.publish(ctx, simPrice, simMsg{Round: m.Round, Value: n.value, TS: m.TS})
	case simPrice:
		n.mu.Lock()
		s := n.get(m.Round)
		if s.priced || s.priceFrom[msg.SentFrom] {
			n.mu.Unlock()
			return nil
		}
		s.priceFrom[msg.SentFrom] = true
		s.prices = append(s.prices, m.Value)
		switch len(s.prices) {
		case 2:
			s.priced = true
			prices := append([]int64(nil), s.prices...)
			n.mu.Unlock()
			return n.fix(ctx, prices, m)
		case 1:
			// like startPriceCollectionTimeout: looks the round up again by ID,
			// so it can act on a newer round that reuses the ID after pruning
			go func() {
				select {
				case <-time.After(n.collect):
				case <-ctx.Done():
					return
				}
				n.mu.Lock()
				s, ok := n.rounds[m.Round]
				if !ok || s.priced || len(s.prices) == 0 {
					n.mu.Unlock()
					return
				}
				s.priced = true
				prices := append([]int64(nil), s.prices...)
				n.mu.Unlock()
				_ = n.fix(ctx, prices, m)
			}()
		}
		n.mu.Unlock()
	case simFix:
		if msg.SentFrom != n.r.GetLeader() {
			return nil
		}
		n.mu.Lock()
		s := n.get(m.Round)
		if s.fixed {
			n.mu.Unlock()
			return nil
		}
		s.fixed = true
		n.mu.Unlock()
		return n.publish(ctx, simProof, m)
	case simProof:
		n.mu.Lock()
		s := n.get(m.Round)
		if s.proved || s.proofFrom[msg.SentFrom] {
			n.mu.Unlock()
			return nil
		}
		s.proofFrom[msg.SentFrom] = true
		s.proofs = append(s.proofs, m)
		switch len(s.proofs) {
		case 2:
			s.proved = true
			n.submit(s.proofs, m)
		case 1:
			// like startProofCollectionTimeout: by ID, with the first message
			go func() {
				select {
				case <-time.After(n.collect):
				case <-ctx.Done():
					return
				}
				n.mu.Lock()
				defer n.mu.Unlock()
				if s, ok := n.rounds[m.Round]; ok && !s.proved && len(s.proofs) > 0 {
					s.proved = true
					n.submit(s.proofs, m)
				}
			}()
		}
		n.mu.Unlock()
	}
	return nil
}

// fix mirrors processCollectedPrices: only a leader publishes the price fix, the
// median of the non-negative prices.
func (n *simNode) fix(ctx context.Context, prices []int64, m simMsg) error {
	if n.r.GetRole() != Leader {
		return nil
	}
	valid := []int64{} // FilterNegative
	for _, p := range prices {
		if p >= 0 {
			valid = append(valid, p)
		}
	}
	if len(valid) == 0 {
		return nil
	}
	median, err := calculator.GetInt64Med(valid)
	if err != nil {
		return err
	}
	return n.publish(ctx, simFix, simMsg{Round: m.Round, Value: median, TS: m.TS})
}

// submit mirrors processCollectedProofs; the DAL checks every proof against
// the submitted value. Caller holds n.mu.
func (n *simNode) submit(proofs []simMsg, m simMsg) {
	n.submissions.Add(1)
	for _, p := range proofs {
		if p.Value != m.Value || !p.TS.Equal(m.TS) {
			n.mixed.Add(1)
			return
		}
	}
}

func newSimNode(ctx context.Context, t *testing.T, scale time.Duration, job time.Duration, value int64) *simNode {
	t.Helper()
	h, err := libp2psetup.NewHost(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = h.Close() })
	ps, err := libp2psetup.MakePubsub(ctx, h)
	require.NoError(t, err)
	topic, err := ps.Join("flap-sim")
	require.NoError(t, err)

	n := &simNode{value: value, collect: 3 * scale / 2, rounds: map[int32]*simRound{}, round: 1}
	n.r = NewRaftNode(h, ps, topic, 1000, job)
	n.r.HeartbeatTimeout = scale
	n.r.CooldownPeriod = 30 * scale
	n.r.LeaderJob = n.leaderJob
	n.r.HandleCustomMessage = n.handleCustom
	return n
}

// forward delivers everything src publishes to dst's Run loop through link, in
// order.
func forward(ctx context.Context, t *testing.T, src, dst *simNode, link *simLink) {
	t.Helper()
	sub, err := src.r.Topic.Subscribe()
	require.NoError(t, err)
	t.Cleanup(sub.Cancel)
	go func() {
		for {
			raw, err := sub.Next(ctx)
			if err != nil {
				return
			}
			// no peers on src's pubsub: everything here is src's own
			select {
			case link.queue <- simQueued{raw: raw, sent: time.Now()}:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case q := <-link.queue:
				// latency from send, so a flushed backlog doesn't pile it up;
				// stall checked on arrival
				time.Sleep(time.Until(q.sent.Add(link.latency)))
				if !link.open() {
					continue
				}
				// into Run's buffer, as subscribe does: it competes with the
				// election timer there
				select {
				case dst.r.MessageBuffer <- q.raw:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func short(id string) string {
	if len(id) > 6 {
		return id[len(id)-6:]
	}
	return id
}

type simTrial struct {
	dualLeader    time.Duration
	mixed         int64
	submissions   int64
	zombie        int64
	leaderless    float64 // share of the trial with no node in Role == Leader
	stuck         bool    // started without a stable leader
	handedOver    bool    // leader at the end isn't the one at the start
	stall         time.Duration
	leaderChanges int64
}

// TestFlapSimulation measures what one flap costs. Statistical and slow, so opt-in:
// RAFT_FLAP_SIM=1 go test -run TestFlapSimulation -v ./pkg/raft/
// Optional: RAFT_FLAP_SIM_TRIALS=<n>, RAFT_FLAP_SIM_MODES=<comma list>,
// RAFT_FLAP_SIM_VERBOSE=1. Modes: {inbound,both}-{queue,drop}[-rand]; inbound
// stalls only the follower's inbound link, both stalls both directions.
// Times are production /10: heartbeat 10 ms, election timer 50-100 ms, cooldown 300 ms.
func TestFlapSimulation(t *testing.T) {
	if os.Getenv("RAFT_FLAP_SIM") == "" {
		t.Skip("set RAFT_FLAP_SIM=1")
	}
	trials := 20
	if v := os.Getenv("RAFT_FLAP_SIM_TRIALS"); v != "" {
		n, err := strconv.Atoi(v)
		require.NoError(t, err, "RAFT_FLAP_SIM_TRIALS")
		require.Positive(t, n, "RAFT_FLAP_SIM_TRIALS")
		trials = n
	}
	modes := []string{"inbound-queue", "both-queue", "inbound-drop", "both-drop"}
	if v := os.Getenv("RAFT_FLAP_SIM_MODES"); v != "" {
		modes = strings.Split(strings.ReplaceAll(v, " ", ""), ",")
	}
	for _, mode := range modes {
		require.Regexp(t, `^(inbound|both)-(queue|drop)(-rand)?$`, mode, "RAFT_FLAP_SIM_MODES")
	}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) { runFlapSim(t, mode, trials) })
	}
}

func runFlapSim(t *testing.T, mode string, trials int) {
	const (
		scale = 10 * time.Millisecond
		job   = 4 * scale
	)
	// "-rand": stall 1.5-6 s (production), so its end races the election
	both := strings.HasPrefix(mode, "both")
	drop := strings.Contains(mode, "drop")
	random := strings.HasSuffix(mode, "-rand")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := newSimNode(ctx, t, scale, job, 100)
	b := newSimNode(ctx, t, scale, job, 102)
	ab := &simLink{latency: scale / 2, drop: drop, queue: make(chan simQueued, 100000)}
	ba := &simLink{latency: scale / 2, drop: drop, queue: make(chan simQueued, 100000)}
	forward(ctx, t, a, b, ab)
	forward(ctx, t, b, a, ba)
	go a.r.Run(ctx)
	time.Sleep(3 * scale)
	go b.r.Run(ctx)

	// role sampler: resolution is one tick (scale/5), and the two roles are read
	// under separate locks, so overlaps of a few ticks are approximate
	var leaderChanges, dual, leaderless atomic.Int64
	go func() {
		tick := time.NewTicker(scale / 5)
		defer tick.Stop()
		prev := [2]RoleType{}
		last := time.Now()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-tick.C:
				// elapsed, not the tick period: a slow receiver drops ticks
				elapsed := now.Sub(last)
				last = now
				ra, rb := a.r.GetRole(), b.r.GetRole()
				if ra == Leader && rb == Leader {
					dual.Add(int64(elapsed))
				}
				if ra != Leader && rb != Leader {
					leaderless.Add(int64(elapsed))
				}
				if ra == Leader && prev[0] != Leader {
					leaderChanges.Add(1)
				}
				if rb == Leader && prev[1] != Leader {
					leaderChanges.Add(1)
				}
				prev = [2]RoleType{ra, rb}
			}
		}
	}()

	// stable waits for one leader both follow. v0.0.27 code can end in an
	// absorbing state (both followers, each following the other, kept there by
	// stray leader loops); the next stall is what breaks it, so the trial goes on.
	stable := func() (in, out *simLink, stuck bool) {
		ok := func() bool {
			ra, rb := a.r.GetRole(), b.r.GetRole()
			return (ra == Leader) != (rb == Leader) && a.r.GetLeader() == b.r.GetLeader() && a.r.GetLeader() != ""
		}
		start := time.Now()
		for !ok() {
			if time.Since(start) > 300*scale {
				if os.Getenv("RAFT_FLAP_SIM_VERBOSE") != "" {
					t.Logf("  no stable leader: a=%s/%d/%s b=%s/%d/%s", a.r.GetRole(), a.r.GetCurrentTerm(), short(a.r.GetLeader()), b.r.GetRole(), b.r.GetCurrentTerm(), short(b.r.GetLeader()))
				}
				return ab, ba, true
			}
			time.Sleep(scale / 2)
		}
		if a.r.GetRole() == Leader {
			return ab, ba, false // follower b's inbound, outbound
		}
		return ba, ab, false
	}

	results := make([]simTrial, 0, trials)
	for i := 0; i < trials; i++ {
		in, out, stuck := stable()
		time.Sleep(50 * scale) // past cooldown, rounds settled
		leader0 := a.r.GetLeader()
		mixed0 := a.mixed.Load() + b.mixed.Load()
		zombie0 := a.zombie.Load() + b.zombie.Load()
		subs0 := a.submissions.Load() + b.submissions.Load()
		// deltas, as the sampler keeps adding
		dual0, leaderless0 := dual.Load(), leaderless.Load()
		trialStart := time.Now()
		changes0 := leaderChanges.Load()

		// follower stalls past its election timeout (2 missed + 1 timer)
		stall := 40 * scale
		if random {
			stall = time.Duration(15+rand.Intn(46)) * scale
		}
		in.stall(stall)
		if both {
			out.stall(stall)
		}
		time.Sleep(150 * scale)
		leader1 := a.r.GetLeader()

		results = append(results, simTrial{
			stall:         stall,
			handedOver:    leader0 != "" && leader1 != "" && leader0 != leader1,
			dualLeader:    time.Duration(dual.Load() - dual0),
			leaderless:    float64(leaderless.Load()-leaderless0) / float64(time.Since(trialStart)),
			stuck:         stuck,
			mixed:         a.mixed.Load() + b.mixed.Load() - mixed0,
			submissions:   a.submissions.Load() + b.submissions.Load() - subs0,
			zombie:        a.zombie.Load() + b.zombie.Load() - zombie0,
			leaderChanges: leaderChanges.Load() - changes0,
		})
	}

	// per trial: what a feed submits across a flap, outages included
	var allMixed, allSubs int64
	var allLeaderless float64
	stuckTrials := 0
	for _, r := range results {
		allMixed += r.mixed
		allSubs += r.submissions
		allLeaderless += r.leaderless
		if r.stuck {
			stuckTrials++
		}
	}
	nt := float64(len(results))
	t.Logf("mode=%s trials=%d | per trial: mixed-proof submissions=%.2f submissions=%.1f leaderless=%.0f%% | started without a stable leader: %d",
		mode, len(results), float64(allMixed)/nt, float64(allSubs)/nt, 100*allLeaderless/nt, stuckTrials)

	// per flap: only trials where leadership changed hands (stall too short for
	// an election otherwise, or the old leader won back)
	var dualSum time.Duration
	var mixedSum, subsSum, zombieSum int64
	duals := []time.Duration{}
	for _, r := range results {
		if !r.handedOver || r.stuck {
			continue
		}
		dualSum += r.dualLeader
		mixedSum += r.mixed
		subsSum += r.submissions
		zombieSum += r.zombie
		duals = append(duals, r.dualLeader)
	}
	flaps := len(duals)
	if flaps == 0 {
		t.Logf("mode=%s trials=%d: no flaps", mode, len(results))
		return
	}
	sort.Slice(duals, func(i, j int) bool { return duals[i] < duals[j] })
	t.Logf("mode=%s trials=%d flaps=%d | per flap: mixed-proof submissions=%.2f submissions=%.1f stray-loop triggers=%.2f | dual-leader mean=%v p50=%v max=%v",
		mode, len(results), flaps, float64(mixedSum)/float64(flaps), float64(subsSum)/float64(flaps), float64(zombieSum)/float64(flaps),
		dualSum/time.Duration(flaps), duals[(flaps-1)/2], duals[flaps-1])
	if os.Getenv("RAFT_FLAP_SIM_VERBOSE") != "" {
		for i, r := range results {
			t.Logf("  trial %2d: stall=%v handed-over=%t stuck=%t mixed=%d subs=%d stray=%d dual=%v changes=%d",
				i, r.stall, r.handedOver, r.stuck, r.mixed, r.submissions, r.zombie, r.dualLeader, r.leaderChanges)
		}
	}
}
