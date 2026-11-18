//nolint:all
package wss

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
)

func countMsgs(msgs []string, want string) int {
	n := 0
	for _, m := range msgs {
		if m == want {
			n++
		}
	}
	return n
}

// writerLog records which conn-bound writer each keepalive call got, in order.
type writerLog struct {
	mu      sync.Mutex
	writers []ConnWriter
}

func (l *writerLog) add(w ConnWriter) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writers = append(l.writers, w)
}

func (l *writerLog) snapshot() []ConnWriter {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]ConnWriter(nil), l.writers...)
}

func TestKeepaliveRunsPerConnection(t *testing.T) {
	rs := newRecordingServer(t, nil)
	var calls writerLog
	ws, err := NewWebsocketHelper(context.Background(),
		WithEndpoint(rs.url()),
		// every cycle = dial + 10ms pre-subscribe wait + 500ms
		WithReconnectInterval(500*time.Millisecond),
		WithKeepalive(50*time.Millisecond, func(ctx context.Context, w ConnWriter) error {
			calls.add(w)
			return w.RawWrite(ctx, "ka")
		}),
	)
	require.NoError(t, err)
	// keepalive also runs during the post-dial wait; keep it short so a
	// conn lives ~ReconnectInterval
	ws.subscribeDelay = 10 * time.Millisecond
	stop := startRun(t, ws, nil)

	conns := rs.waitDials(4, 10*time.Second)
	stop()
	require.GreaterOrEqual(t, len(conns), 4, "expected >= 3 reconnects")

	// every finished connection got keepalives at roughly the interval
	// (500ms / 50ms = ~10)
	for i := 0; i < 3; i++ {
		got := countMsgs(conns[i].msgs, "ka")
		assert.GreaterOrEqual(t, got, 5, "conn %d keepalives", i)
		assert.LessOrEqual(t, got, 12, "conn %d keepalives", i)
	}

	// one loop per connection: once a new conn's writer shows up, the
	// previous conn's loop is never called again
	writers := calls.snapshot()
	seen := map[ConnWriter]bool{}
	var last ConnWriter
	for _, w := range writers {
		if w != last {
			assert.False(t, seen[w], "keepalive of an ended connection called again")
			seen[w] = true
			last = w
		}
	}
	assert.GreaterOrEqual(t, len(seen), 3)

	// nothing keeps ticking after Run returned
	n := len(calls.snapshot())
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, n, len(calls.snapshot()), "keepalive kept running after Run returned")
}

func TestKeepaliveSurvivesFailedWrites(t *testing.T) {
	rs := newRecordingServer(t, nil)
	var calls atomic.Int64
	ws, err := NewWebsocketHelper(context.Background(),
		WithEndpoint(rs.url()),
		WithKeepalive(20*time.Millisecond, func(ctx context.Context, w ConnWriter) error {
			// fail every other call: never MaxKeepaliveFailures in a row
			if calls.Add(1)%2 == 1 {
				return errors.New("transient")
			}
			return w.RawWrite(ctx, "ka")
		}),
	)
	require.NoError(t, err)
	startRun(t, ws, nil)

	require.Eventually(t, func() bool { return calls.Load() >= 20 }, 5*time.Second, 10*time.Millisecond, "keepalive stopped after a failure")
	s := rs.snapshot()
	assert.Len(t, s, 1, "isolated failures must not reconnect")
	assert.GreaterOrEqual(t, countMsgs(s[0].msgs, "ka"), 5)
}

func TestKeepaliveConsecutiveFailuresReconnect(t *testing.T) {
	rs := newRecordingServer(t, nil)
	var mu sync.Mutex
	perWriter := map[ConnWriter]int{}
	ws, err := NewWebsocketHelper(context.Background(),
		WithEndpoint(rs.url()),
		WithKeepalive(50*time.Millisecond, func(ctx context.Context, w ConnWriter) error {
			mu.Lock()
			perWriter[w]++
			mu.Unlock()
			return errors.New("always fails")
		}),
	)
	require.NoError(t, err)
	stop := startRun(t, ws, nil)

	// default inactivity / reconnect windows are minutes / hours, so only the
	// keepalive can end the first conn
	conns := rs.waitDials(2, 5*time.Second)
	stop()
	require.GreaterOrEqual(t, len(conns), 2, "failing keepalive did not force a reconnect")
	assert.Less(t, conns[1].accepted.Sub(conns[0].accepted), 3*time.Second)

	mu.Lock()
	defer mu.Unlock()
	for _, n := range perWriter {
		assert.LessOrEqual(t, n, MaxKeepaliveFailures, "keepalive kept running past MaxKeepaliveFailures")
	}
}

func TestKeepaliveIntervalPerConnection(t *testing.T) {
	rs := newRecordingServer(t, nil)
	var conn atomic.Int64
	intervals := []time.Duration{40 * time.Millisecond, 400 * time.Millisecond, 0}
	ws, err := NewWebsocketHelper(context.Background(),
		WithEndpoint(rs.url()),
		WithReconnectInterval(time.Second),
		WithKeepalive(100*time.Millisecond, func(ctx context.Context, w ConnWriter) error {
			return w.RawWrite(ctx, "ka")
		}),
		WithKeepaliveIntervalFunc(func() time.Duration {
			i := conn.Add(1) - 1
			if int(i) < len(intervals) {
				return intervals[i]
			}
			return 0
		}),
	)
	require.NoError(t, err)
	ws.subscribeDelay = 10 * time.Millisecond
	stop := startRun(t, ws, nil)

	conns := rs.waitDials(4, 10*time.Second)
	stop()
	require.GreaterOrEqual(t, len(conns), 4)

	// 1s windows: 40ms -> ~25, 400ms -> ~2, 0 -> falls back to 100ms -> ~10
	fast := countMsgs(conns[0].msgs, "ka")
	slow := countMsgs(conns[1].msgs, "ka")
	fallback := countMsgs(conns[2].msgs, "ka")
	assert.GreaterOrEqual(t, fast, 15, "conn 0 (40ms)")
	assert.LessOrEqual(t, slow, 3, "conn 1 (400ms)")
	assert.GreaterOrEqual(t, slow, 1, "conn 1 (400ms)")
	assert.GreaterOrEqual(t, fallback, 6, "conn 2 (fallback 100ms)")
	assert.LessOrEqual(t, fallback, 12, "conn 2 (fallback 100ms)")
}

func TestNewWebsocketHelperRejectsBadKeepalive(t *testing.T) {
	fn := func(context.Context, ConnWriter) error { return nil }
	for name, opt := range map[string]ConnectionOption{
		"zero interval":          WithKeepalive(0, fn),
		"negative interval":      WithKeepalive(-time.Second, fn),
		"nil func":               WithKeepalive(time.Second, nil),
		"zero interval, nil func": WithKeepalive(0, nil),
		"interval func w/o func": WithKeepaliveIntervalFunc(func() time.Duration { return time.Second }),
	} {
		t.Run(name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				_, err := NewWebsocketHelper(context.Background(), WithEndpoint("ws://x"), opt)
				assert.Error(t, err)
			})
		})
	}
}

// A caller ctx that ends while a write is in flight must not close the conn
// (nhooyr.io/websocket closes the whole conn when the write ctx ends).
func TestWriteSurvivesCallerCancelMidWrite(t *testing.T) {
	release := make(chan struct{})
	received := make(chan string, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		conn.SetReadLimit(128 << 20)
		// stop reading so the client's big write blocks on full TCP buffers
		<-release
		for {
			_, data, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			if len(data) > 16 {
				received <- "big"
			} else {
				received <- string(data)
			}
		}
	}))
	defer server.Close()
	defer close(release)

	ws, err := NewWebsocketHelper(context.Background(), WithEndpoint("ws"+server.URL[len("http"):]))
	require.NoError(t, err)
	require.NoError(t, ws.Dial(context.Background()))
	defer ws.Close()

	callerCtx, cancel := context.WithCancel(context.Background())
	writeErr := make(chan error, 1)
	big := strings.Repeat("x", 12<<20)
	go func() { writeErr <- ws.RawWrite(callerCtx, big) }()

	time.Sleep(300 * time.Millisecond)
	select {
	case err := <-writeErr:
		t.Fatalf("write was expected to block, returned %v", err)
	default:
	}
	cancel()
	time.Sleep(100 * time.Millisecond)
	release <- struct{}{}

	select {
	case err := <-writeErr:
		assert.NoError(t, err, "cancelled caller ctx aborted the write")
	case <-time.After(WriteTimeout):
		t.Fatal("write did not finish")
	}
	require.NoError(t, ws.RawWrite(context.Background(), "after"), "conn closed by caller ctx cancel")
	for _, want := range []string{"big", "after"} {
		select {
		case got := <-received:
			assert.Equal(t, want, got)
		case <-time.After(5 * time.Second):
			t.Fatalf("server did not receive %q", want)
		}
	}
}

// #2906: providers send one subscribe frame per feed with a pause after each,
// so the keepalive must already run while subscribing.
func TestKeepaliveRunsDuringSubscribe(t *testing.T) {
	rs := newRecordingServer(t, nil)
	const subs = 20
	subscriptions := make([]any, subs)
	for i := range subscriptions {
		subscriptions[i] = []byte(fmt.Sprintf("sub-%d", i))
	}
	ws, err := NewWebsocketHelper(context.Background(),
		WithEndpoint(rs.url()),
		WithSubscriptions(subscriptions),
		WithKeepalive(30*time.Millisecond, func(ctx context.Context, w ConnWriter) error {
			return w.RawWrite(ctx, "ka")
		}),
	)
	require.NoError(t, err)
	// ~1s of subscribing: 50ms after dial + 50ms after each frame
	ws.subscribeDelay = 50 * time.Millisecond
	stop := startRun(t, ws, nil)

	last := fmt.Sprintf("sub-%d", subs-1)
	require.Eventually(t, func() bool {
		s := rs.snapshot()
		return len(s) > 0 && contains(s[0].msgs, last)
	}, 5*time.Second, 10*time.Millisecond, "subscribe did not finish")
	stop()

	msgs := rs.snapshot()[0].msgs
	kaBeforeFirst, kaBeforeLast := 0, 0
	seenFirst := false
	for _, m := range msgs {
		if m == "sub-0" {
			seenFirst = true
		}
		if m == last {
			break
		}
		if m == "ka" {
			kaBeforeLast++
			if !seenFirst {
				kaBeforeFirst++
			}
		}
	}
	assert.GreaterOrEqual(t, kaBeforeFirst, 1, "no keepalive between dial and first subscribe")
	assert.GreaterOrEqual(t, kaBeforeLast, 15, "keepalive not sent during subscribe")
}

// A failed subscribe ends the keepalive started for that conn: no loop
// survives into the next attempt.
func TestKeepaliveStopsAfterFailedSubscribe(t *testing.T) {
	rs := newRecordingServer(t, nil)
	var calls writerLog
	ws, err := NewWebsocketHelper(context.Background(),
		WithEndpoint(rs.url()),
		// not JSON-encodable: every subscribe attempt fails before writing
		WithSubscriptions([]any{make(chan int)}),
		WithKeepalive(10*time.Millisecond, func(ctx context.Context, w ConnWriter) error {
			calls.add(w)
			return w.RawWrite(ctx, "ka")
		}),
	)
	require.NoError(t, err)
	ws.subscribeDelay = 50 * time.Millisecond
	stop := startRun(t, ws, nil)

	conns := rs.waitDials(4, 10*time.Second)
	stop()
	require.GreaterOrEqual(t, len(conns), 4, "expected >= 3 failed subscribes")

	writers := calls.snapshot()
	seen := map[ConnWriter]bool{}
	var last ConnWriter
	for _, w := range writers {
		if w != last {
			assert.False(t, seen[w], "keepalive of a failed-subscribe conn called again")
			seen[w] = true
			last = w
		}
	}
	assert.GreaterOrEqual(t, len(seen), 3, "keepalive did not run during subscribe attempts")

	n := len(calls.snapshot())
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, n, len(calls.snapshot()), "keepalive kept running after Run returned")
}
