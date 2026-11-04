package aggregator

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	libp2pSetup "bisonai.com/miko/node/pkg/libp2p/setup"
	"bisonai.com/miko/node/pkg/nodestate"
	"bisonai.com/miko/node/pkg/raft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openRoundState(t *testing.T, path string) *nodestate.Store {
	t.Helper()
	st, err := nodestate.Open(context.Background(), path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func storedRound(t *testing.T, st *nodestate.Store, name string) int32 {
	t.Helper()
	r, _, err := st.LatestRound(context.Background(), name)
	require.NoError(t, err)
	return r
}

// newRoundTestAggregator builds an aggregator on a local libp2p host (no peers).
func newRoundTestAggregator(t *testing.T, ctx context.Context, name string, rounds *roundPersister) *Aggregator {
	t.Helper()
	h, err := libp2pSetup.NewHost(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = h.Close() })
	ps, err := libp2pSetup.MakePubsub(ctx, h)
	require.NoError(t, err)
	agg, err := NewAggregator(h, ps, name+"-round-test-topic", Config{ID: 1, Name: name, AggregateInterval: 400}, nil, NewLatestLocalAggregates(), rounds)
	require.NoError(t, err)
	return agg
}

// Restart: rounds persisted by one process are continued (N -> N+1) by the next.
func TestRoundRestartContinuesAfterStored(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "node.db")

	st := openRoundState(t, path)
	agg := newRoundTestAggregator(t, ctx, "BTC-USDT", newRoundPersister(st))
	require.NoError(t, agg.loadLatestRound(ctx))
	require.Equal(t, int32(1), agg.RoundID, "fresh store starts at 1")

	agg.RoundID = 41
	require.NoError(t, agg.LeaderJob(ctx)) // leader advances to 42
	require.Equal(t, int32(42), agg.RoundID)
	require.NoError(t, agg.rounds.Flush(ctx))
	require.NoError(t, st.Close())

	st2 := openRoundState(t, path)
	agg2 := newRoundTestAggregator(t, ctx, "BTC-USDT", newRoundPersister(st2))
	require.NoError(t, agg2.loadLatestRound(ctx))
	assert.Equal(t, int32(43), agg2.RoundID)

	require.NoError(t, agg2.LeaderJob(ctx))
	assert.Equal(t, int32(44), agg2.RoundID, "first leader round after restart is stored+2")
}

// Follower: max(trigger, own) is persisted.
func TestRoundFollowerPersistsMaxOfTriggerAndOwn(t *testing.T) {
	ctx := context.Background()
	st := openRoundState(t, filepath.Join(t.TempDir(), "node.db"))
	agg := newRoundTestAggregator(t, ctx, "ETH-USDT", newRoundPersister(st))
	agg.RoundID = 10
	agg.Raft.LeaderID = "leader-peer"

	trigger := func(round int32) {
		data, err := encodeInner(TriggerMessage{LeaderID: "leader-peer", RoundID: round, Timestamp: time.Now()})
		require.NoError(t, err)
		require.NoError(t, agg.HandleTriggerMessage(ctx, raft.Message{Type: Trigger, SentFrom: "leader-peer", Data: data}))
	}

	trigger(57)
	assert.Equal(t, int32(57), agg.RoundID)
	require.NoError(t, agg.rounds.Flush(ctx))
	assert.Equal(t, int32(57), storedRound(t, st, "ETH-USDT"))

	// a stale leader's lower round keeps our own
	trigger(30)
	assert.Equal(t, int32(57), agg.RoundID)
	require.NoError(t, agg.rounds.Flush(ctx))
	assert.Equal(t, int32(57), storedRound(t, st, "ETH-USDT"))
}

// The writer loop persists Advance()d rounds without an explicit Flush and
// flushes the rest when stopped.
func TestRoundPersisterRunWritesAsync(t *testing.T) {
	st := openRoundState(t, filepath.Join(t.TempDir(), "node.db"))
	p := newRoundPersister(st)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { p.run(ctx); close(done) }()

	p.Advance("A", 5)
	assert.Eventually(t, func() bool { return storedRound(t, st, "A") == 5 }, 2*time.Second, 10*time.Millisecond)

	p.Advance("A", 3) // lower: ignored
	p.Advance("B", 9)
	cancel()
	<-done
	assert.Equal(t, int32(5), storedRound(t, st, "A"))
	assert.Equal(t, int32(9), storedRound(t, st, "B"))
}

// Latest prefers the in-memory value over a lagging store, so a refresh never
// rewinds an aggregator.
func TestRoundPersisterLatestUsesMemory(t *testing.T) {
	ctx := context.Background()
	st := openRoundState(t, filepath.Join(t.TempDir(), "node.db"))
	require.NoError(t, st.SaveRounds(ctx, map[string]int32{"A": 100}))

	p := newRoundPersister(st)
	r, err := p.Latest(ctx, "A")
	require.NoError(t, err)
	assert.Equal(t, int32(100), r)

	p.Advance("A", 105) // not flushed
	r, err = p.Latest(ctx, "A")
	require.NoError(t, err)
	assert.Equal(t, int32(105), r)

	r, err = p.Latest(ctx, "missing")
	require.NoError(t, err)
	assert.Equal(t, int32(0), r)
}

type failingRoundStore struct {
	mu        sync.Mutex
	fail      bool
	readFails int // LatestRound fails this many times, then returns 77
	saved     map[string]int32
}

func (f *failingRoundStore) LatestRound(context.Context, string) (int32, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.readFails > 0 {
		f.readFails--
		return 0, false, errors.New("database is locked")
	}
	return 77, true, nil
}

func (f *failingRoundStore) SaveRounds(_ context.Context, rounds map[string]int32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return errors.New("disk full")
	}
	for k, v := range rounds {
		f.saved[k] = v
	}
	return nil
}

// A failed write keeps the rounds pending for the next flush.
func TestRoundPersisterFlushRetriesAfterError(t *testing.T) {
	ctx := context.Background()
	store := &failingRoundStore{fail: true, saved: map[string]int32{}}
	p := newRoundPersister(store)

	p.Advance("A", 7)
	assert.Error(t, p.Flush(ctx))
	p.Advance("A", 8)

	store.fail = false
	require.NoError(t, p.Flush(ctx))
	assert.Equal(t, map[string]int32{"A": 8}, store.saved)
}

// A failed read of the stored round is retried, never falls back to round 1.
func TestLoadLatestRoundRetriesReadError(t *testing.T) {
	orig := loadLatestRoundRetryDelay
	loadLatestRoundRetryDelay = time.Millisecond
	defer func() { loadLatestRoundRetryDelay = orig }()

	store := &failingRoundStore{readFails: 3, saved: map[string]int32{}}
	agg := &Aggregator{Config: Config{Name: "A"}, RoundID: 1, rounds: newRoundPersister(store)}
	require.NoError(t, agg.loadLatestRound(context.Background()))
	assert.Equal(t, int32(78), agg.RoundID)

	// gives up only when ctx is done, leaving RoundID untouched
	store.readFails = 1 << 30
	agg2 := &Aggregator{Config: Config{Name: "B"}, RoundID: 1, rounds: newRoundPersister(store)}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	assert.Error(t, agg2.loadLatestRound(ctx))
	assert.Equal(t, int32(1), agg2.RoundID)
}
