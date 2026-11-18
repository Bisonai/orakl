//nolint:all
package wss_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/wss"
	"bisonai.com/miko/node/pkg/wss/wsstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
)

type pingConn struct {
	accepted time.Time
	pings    atomic.Int64
	mu       sync.Mutex
	// text frames received, each with the ping count when it arrived
	frames []string
	pingAt []int64
}

// pingServer accepts websockets, counts the client's pings per connection
// and records its text frames. If read is false it never reads, so pings
// are never answered.
type pingServer struct {
	*httptest.Server
	mu    sync.Mutex
	conns []*pingConn
}

func newPingServer(t *testing.T, read bool) *pingServer {
	t.Helper()
	ps := &pingServer{}
	// the request ctx is not cancelled once the conn is hijacked
	done := make(chan struct{})
	ps.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pc := &pingConn{accepted: time.Now()}
		conn, err := websocket.Accept(wsstest.CountPings(w, &pc.pings), r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		ps.mu.Lock()
		ps.conns = append(ps.conns, pc)
		ps.mu.Unlock()

		if !read {
			<-done
			return
		}
		for {
			_, data, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			pc.mu.Lock()
			pc.frames = append(pc.frames, string(data))
			pc.pingAt = append(pc.pingAt, pc.pings.Load())
			pc.mu.Unlock()
		}
	}))
	t.Cleanup(ps.Close)
	// runs before ps.Close (cleanups are LIFO)
	t.Cleanup(func() { close(done) })
	return ps
}

func (ps *pingServer) url() string { return "ws" + ps.URL[len("http"):] }

func (ps *pingServer) snapshot() []*pingConn {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return append([]*pingConn(nil), ps.conns...)
}

func (ps *pingServer) waitConns(n int, timeout time.Duration) []*pingConn {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if s := ps.snapshot(); len(s) >= n {
			return s
		}
		time.Sleep(10 * time.Millisecond)
	}
	return ps.snapshot()
}

// runHelper runs ws in the background; the returned stop cancels it and
// fails the test unless Run returns within stopTimeout.
func runHelper(t *testing.T, ws *wss.WebsocketHelper, stopTimeout time.Duration) func() {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		ws.Run(ctx, func(context.Context, map[string]any) error { return nil })
		close(done)
	}()
	stop := func() {
		cancel()
		select {
		case <-done:
		case <-time.After(stopTimeout):
			t.Fatal("Run did not return promptly after context cancel")
		}
	}
	t.Cleanup(stop)
	return stop
}

func TestPingKeepalivePerConnection(t *testing.T) {
	ps := newPingServer(t, true)
	ws, err := wss.NewWebsocketHelper(context.Background(),
		wss.WithEndpoint(ps.url()),
		wss.WithReconnectInterval(500*time.Millisecond),
		wss.WithPingKeepalive(30*time.Millisecond),
	)
	require.NoError(t, err)
	wss.SetSubscribeDelay(ws, 10*time.Millisecond)
	stop := runHelper(t, ws, 2*time.Second)

	conns := ps.waitConns(4, 10*time.Second)
	stop()
	require.GreaterOrEqual(t, len(conns), 4, "expected >= 3 reconnects")

	for i := 0; i < 3; i++ {
		// ~510ms per conn / 30ms
		got := conns[i].pings.Load()
		assert.GreaterOrEqual(t, got, int64(8), "conn %d pings", i)
		assert.LessOrEqual(t, got, int64(25), "conn %d pings", i)
		// answered pings never force a reconnect: conns last the full interval
		gap := conns[i+1].accepted.Sub(conns[i].accepted)
		assert.GreaterOrEqual(t, gap, 450*time.Millisecond, "conn %d ended early", i)
	}
}

// Pongs are only read once Run's read loop starts, so pings sent while
// subscribing time out; they must not count as keepalive failures.
func TestPingKeepaliveDuringSubscribe(t *testing.T) {
	ps := newPingServer(t, true)
	const subs = 10
	subscriptions := make([]any, subs)
	for i := range subscriptions {
		subscriptions[i] = []byte(fmt.Sprintf("sub-%d", i))
	}
	ws, err := wss.NewWebsocketHelper(context.Background(),
		wss.WithEndpoint(ps.url()),
		wss.WithSubscriptions(subscriptions),
		wss.WithPingKeepalive(20*time.Millisecond),
	)
	require.NoError(t, err)
	// ~550ms of subscribing, > MaxKeepaliveFailures ping timeouts
	wss.SetSubscribeDelay(ws, 50*time.Millisecond)
	stop := runHelper(t, ws, 2*time.Second)

	last := fmt.Sprintf("sub-%d", subs-1)
	require.Eventually(t, func() bool {
		s := ps.snapshot()
		if len(s) == 0 {
			return false
		}
		s[0].mu.Lock()
		defer s[0].mu.Unlock()
		return len(s[0].frames) == subs
	}, 5*time.Second, 10*time.Millisecond, "subscribe did not finish")
	// read loop runs now; pings keep being answered
	time.Sleep(300 * time.Millisecond)
	stop()

	conns := ps.snapshot()
	assert.Len(t, conns, 1, "pings while subscribing forced a reconnect")
	c := conns[0]
	c.mu.Lock()
	defer c.mu.Unlock()
	require.Equal(t, last, c.frames[subs-1])
	assert.GreaterOrEqual(t, c.pingAt[subs-1], int64(5), "no pings while subscribing")
	assert.Greater(t, c.pings.Load(), c.pingAt[subs-1]+5, "pings stopped after subscribe")
}

func TestPingKeepaliveMissingPongReconnects(t *testing.T) {
	// the server never reads, so pings are never answered
	ps := newPingServer(t, false)
	ws, err := wss.NewWebsocketHelper(context.Background(),
		wss.WithEndpoint(ps.url()),
		wss.WithPingKeepalive(30*time.Millisecond),
	)
	require.NoError(t, err)
	wss.SetSubscribeDelay(ws, 10*time.Millisecond)
	// closing waits up to 5s for the close handshake of a peer that never reads
	stop := runHelper(t, ws, 8*time.Second)

	// default inactivity / reconnect windows are minutes / hours, so only
	// the ping keepalive can end the first conn
	conns := ps.waitConns(2, 5*time.Second)
	stop()
	require.GreaterOrEqual(t, len(conns), 2, "unanswered pings did not force a reconnect")
	assert.Less(t, conns[1].accepted.Sub(conns[0].accepted), 3*time.Second)
}

func TestPingKeepaliveRejectsBadInterval(t *testing.T) {
	for _, d := range []time.Duration{0, -time.Second} {
		_, err := wss.NewWebsocketHelper(context.Background(), wss.WithEndpoint("ws://x"), wss.WithPingKeepalive(d))
		assert.Error(t, err, "interval %v", d)
	}
}
