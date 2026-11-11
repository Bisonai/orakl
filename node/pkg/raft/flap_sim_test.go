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
	msg  Message
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
	n.mu.Lock()
	n.round++
	round := n.round
	n.mu.Unlock()
	return n.publish(ctx, simTrigger, simMsg{Round: round, TS: time.Now()})
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
			n.mu.Unlock()
			return n.fix(ctx, s.prices, m)
		case 1:
			go func() {
				time.Sleep(n.collect)
				n.mu.Lock()
				if s.priced {
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
			go func() {
				time.Sleep(n.collect)
				n.mu.Lock()
				defer n.mu.Unlock()
				if !s.proved {
					s.proved = true
					n.submit(s.proofs, m)
				}
			}()
		}
		n.mu.Unlock()
	}
	return nil
}

// fix mirrors processCollectedPrices: only a leader publishes the price fix.
func (n *simNode) fix(ctx context.Context, prices []int64, m simMsg) error {
	if n.r.GetRole() != Leader {
		return nil
	}
	sum := int64(0)
	for _, p := range prices {
		sum += p
	}
	return n.publish(ctx, simFix, simMsg{Round: m.Round, Value: sum / int64(len(prices)), TS: m.TS})
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

// forward delivers everything src publishes to dst through link, in order.
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
			msg, err := src.r.unmarshalMessage(raw.Data)
			if err != nil || msg.SentFrom != src.r.GetHostId() {
				continue
			}
			select {
			case link.queue <- simQueued{msg: msg, sent: time.Now()}:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case q := <-link.queue:
				if !link.open() {
					continue
				}
				// latency from send, so a flushed backlog doesn't pile it up
				time.Sleep(time.Until(q.sent.Add(link.latency)))
				go func() { _ = dst.r.handleMessage(ctx, q.msg) }()
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

func roleOf(n *simNode) RoleType { return n.r.GetRole() }

type simTrial struct {
	dualLeader    time.Duration
	mixed         int64
	submissions   int64
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
	if v, err := strconv.Atoi(os.Getenv("RAFT_FLAP_SIM_TRIALS")); err == nil {
		trials = v
	}
	modes := []string{"inbound-queue", "both-queue", "inbound-drop", "both-drop"}
	if v := os.Getenv("RAFT_FLAP_SIM_MODES"); v != "" {
		modes = strings.Split(v, ",")
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

	var leaderChanges, dual atomic.Int64
	go func() {
		tick := time.NewTicker(scale / 5)
		defer tick.Stop()
		prev := [2]RoleType{}
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				ra, rb := roleOf(a), roleOf(b)
				if ra == Leader && rb == Leader {
					dual.Add(int64(scale / 5))
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

	stable := func() (in, out *simLink) {
		ok := func() bool {
			ra, rb := roleOf(a), roleOf(b)
			return (ra == Leader) != (rb == Leader) && a.r.GetLeader() == b.r.GetLeader() && a.r.GetLeader() != ""
		}
		start := time.Now()
		for !ok() {
			if time.Since(start) > 2000*scale {
				t.Fatalf("no stable leader: a=%s/%d/%s b=%s/%d/%s", roleOf(a), a.r.GetCurrentTerm(), short(a.r.GetLeader()), roleOf(b), b.r.GetCurrentTerm(), short(b.r.GetLeader()))
			}
			time.Sleep(scale / 2)
		}
		if roleOf(a) == Leader {
			return ab, ba // follower b's inbound, outbound
		}
		return ba, ab
	}

	results := make([]simTrial, 0, trials)
	for i := 0; i < trials; i++ {
		in, out := stable()
		time.Sleep(50 * scale) // past cooldown, rounds settled
		mixed0 := a.mixed.Load() + b.mixed.Load()
		subs0 := a.submissions.Load() + b.submissions.Load()
		dual.Store(0)
		leaderChanges.Store(0)

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

		results = append(results, simTrial{
			dualLeader:    time.Duration(dual.Load()),
			mixed:         a.mixed.Load() + b.mixed.Load() - mixed0,
			submissions:   a.submissions.Load() + b.submissions.Load() - subs0,
			leaderChanges: leaderChanges.Load(),
		})
	}

	var dualSum time.Duration
	var mixedSum, subsSum, changeSum int64
	duals := make([]time.Duration, 0, len(results))
	for _, r := range results {
		dualSum += r.dualLeader
		mixedSum += r.mixed
		subsSum += r.submissions
		changeSum += r.leaderChanges
		duals = append(duals, r.dualLeader)
	}
	sort.Slice(duals, func(i, j int) bool { return duals[i] < duals[j] })
	n := len(results)
	t.Logf("mode=%s trials=%d | mixed-proof submissions/flap=%.2f | submissions/flap=%.1f | dual-leader mean=%v median=%v max=%v | leader changes/flap=%.2f",
		mode, n, float64(mixedSum)/float64(n), float64(subsSum)/float64(n),
		dualSum/time.Duration(n), duals[n/2], duals[n-1], float64(changeSum)/float64(n))
	if os.Getenv("RAFT_FLAP_SIM_VERBOSE") != "" {
		for i, r := range results {
			t.Logf("  trial %2d: mixed=%d subs=%d dual=%v changes=%d", i, r.mixed, r.submissions, r.dualLeader, r.leaderChanges)
		}
	}
}
