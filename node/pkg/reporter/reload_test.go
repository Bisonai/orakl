package reporter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// fakeDalWs records the SUBSCRIBE params of every connection and counts open ones.
type fakeDalWs struct {
	mu   sync.Mutex
	subs [][]string
	open atomic.Int32
}

func (f *fakeDalWs) lastSubs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.subs) == 0 {
		return nil
	}
	out := append([]string(nil), f.subs[len(f.subs)-1]...)
	sort.Strings(out)
	return out
}

func newFakeDalWs(t *testing.T) (*fakeDalWs, string) {
	t.Helper()
	f := &fakeDalWs{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		f.open.Add(1)
		defer f.open.Add(-1)
		for {
			var msg Subscription
			if err := wsjson.Read(r.Context(), c, &msg); err != nil {
				return
			}
			f.mu.Lock()
			f.subs = append(f.subs, msg.Params)
			f.mu.Unlock()
		}
	}))
	t.Cleanup(srv.Close)
	return f, "ws" + strings.TrimPrefix(srv.URL, "http")
}

func interval(ms int) *int { return &ms }

// newReloadApp builds reporters for configs without chain or DAL REST access
// (long intervals, so no job runs during the test).
func newReloadApp(t *testing.T, ctx context.Context, wsURL string, configs []Config) *App {
	t.Helper()
	a := New()
	a.deps = reporterDeps{dalWsEndpoint: wsURL, dalApiKey: "key", dalRestEndpoint: "http://127.0.0.1:1"}
	a.mu.Lock()
	require.NoError(t, a.buildReporters(ctx, configs))
	a.mu.Unlock()
	a.startReporters(ctx)
	return a
}

func groupPairs(a *App) map[int][]string {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := map[int][]string{}
	for i, r := range a.groups {
		p := r.GetPairs()
		sort.Strings(p)
		out[i] = p
	}
	return out
}

func TestApplyConfigsDiff(t *testing.T) {
	origGrace := wsSwapGrace
	wsSwapGrace = 50 * time.Millisecond
	t.Cleanup(func() { wsSwapGrace = origGrace })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ws, wsURL := newFakeDalWs(t)

	a := newReloadApp(t, ctx, wsURL, []Config{
		{Name: "A-B", SubmitInterval: interval(60000)},
		{Name: "C-D", SubmitInterval: interval(60000)},
		{Name: "E-F", SubmitInterval: interval(120000)},
	})
	require.Eventually(t, func() bool { return len(ws.lastSubs()) == 3 }, 5*time.Second, 20*time.Millisecond)
	oldGroup := a.groups[60000]
	stoppedGroup := a.groups[120000]

	a.LatestSubmittedDataMap.Store("E-F", int64(1))
	a.LatestDataMap.Store("E-F", SubmissionData{})

	// add G-H (new interval), remove E-F (its interval goes away), keep A-B, C-D
	err := a.applyConfigs(ctx, []Config{
		{Name: "A-B", SubmitInterval: interval(60000)},
		{Name: "C-D", SubmitInterval: interval(60000)},
		{Name: "G-H", SubmitInterval: interval(180000)},
	})
	require.NoError(t, err)

	assert.Equal(t, map[int][]string{60000: {"A-B", "C-D"}, 180000: {"G-H"}}, groupPairs(a))
	assert.Same(t, oldGroup, a.groups[60000], "unchanged group must be kept, not recreated")
	select {
	case <-stoppedGroup.stop:
	default:
		t.Fatal("group without pairs must be stopped")
	}
	dev := a.deviation.GetPairs()
	sort.Strings(dev)
	assert.Equal(t, []string{"A-B", "C-D", "G-H"}, dev)
	assert.Len(t, a.Reporters, 3)

	_, ok := a.LatestSubmittedDataMap.Load("E-F")
	assert.False(t, ok)
	_, ok = a.LatestDataMap.Load("E-F")
	assert.False(t, ok)

	// new websocket subscribed to the new set; old one closed after the grace
	require.Eventually(t, func() bool {
		s := ws.lastSubs()
		return len(s) == 3 && s[2] == "submission@G-H"
	}, 5*time.Second, 20*time.Millisecond)
	require.Eventually(t, func() bool { return ws.open.Load() == 1 }, 5*time.Second, 20*time.Millisecond)

	// move C-D to the 180s group
	err = a.applyConfigs(ctx, []Config{
		{Name: "A-B", SubmitInterval: interval(60000)},
		{Name: "C-D", SubmitInterval: interval(180000)},
		{Name: "G-H", SubmitInterval: interval(180000)},
	})
	require.NoError(t, err)
	assert.Equal(t, map[int][]string{60000: {"A-B"}, 180000: {"C-D", "G-H"}}, groupPairs(a))

	// same content: no-op (no new websocket)
	ws.mu.Lock()
	subsBefore := len(ws.subs)
	ws.mu.Unlock()
	require.NoError(t, a.applyConfigs(ctx, []Config{
		{Name: "A-B", SubmitInterval: interval(60000)},
		{Name: "C-D", SubmitInterval: interval(180000)},
		{Name: "G-H", SubmitInterval: interval(180000)},
	}))
	time.Sleep(100 * time.Millisecond)
	ws.mu.Lock()
	assert.Equal(t, subsBefore, len(ws.subs))
	ws.mu.Unlock()
}

func TestPruneSubmitted(t *testing.T) {
	r := &Reporter{LatestSubmittedDataMap: new(sync.Map), Pairs: []string{"A-B"}}
	r.LatestSubmittedDataMap.Store("A-B", int64(1))
	r.LatestSubmittedDataMap.Store("REMOVED", int64(1))
	r.pruneSubmitted()
	_, ok := r.LatestSubmittedDataMap.Load("A-B")
	assert.True(t, ok)
	_, ok = r.LatestSubmittedDataMap.Load("REMOVED")
	assert.False(t, ok)
}

func TestReporterStopEndsRun(t *testing.T) {
	r, err := NewReporter(context.Background(), WithConfigs([]Config{{Name: "A-B"}}), WithInterval(60000))
	require.NoError(t, err)
	done := make(chan struct{})
	go func() {
		r.Run(context.Background())
		close(done)
	}()
	r.Stop()
	r.Stop() // idempotent
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after Stop")
	}
}

// DAL may learn an added pair later than the reporter and ignores its
// SUBSCRIBE until then: the reporter resubscribes until data arrives.
func TestResubscribeUntilAddedPairHasData(t *testing.T) {
	origGrace, origDelay := wsSwapGrace, wsResubscribeDelay
	wsSwapGrace, wsResubscribeDelay = 20*time.Millisecond, 1500*time.Millisecond // > dial + 1s subscribe delay
	t.Cleanup(func() { wsSwapGrace, wsResubscribeDelay = origGrace, origDelay })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ws, wsURL := newFakeDalWs(t)
	count := func() int {
		ws.mu.Lock()
		defer ws.mu.Unlock()
		return len(ws.subs)
	}

	a := newReloadApp(t, ctx, wsURL, []Config{{Name: "A-B", SubmitInterval: interval(60000)}})
	require.Eventually(t, func() bool { return count() == 1 }, 5*time.Second, 20*time.Millisecond)

	require.NoError(t, a.applyConfigs(ctx, []Config{
		{Name: "A-B", SubmitInterval: interval(60000)},
		{Name: "C-D", SubmitInterval: interval(60000)},
	}))
	// reload swap + at least one resubscribe while C-D has no data
	require.Eventually(t, func() bool { return count() >= 3 }, 10*time.Second, 20*time.Millisecond)

	a.LatestDataMap.Store("C-D", SubmissionData{})
	time.Sleep(3 * wsResubscribeDelay) // let a pending resubscribe fire
	settled := count()
	time.Sleep(3 * wsResubscribeDelay)
	assert.Equal(t, settled, count(), "no resubscribe once data arrives")
}
