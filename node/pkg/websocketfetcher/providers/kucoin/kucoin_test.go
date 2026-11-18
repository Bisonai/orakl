package kucoin

import (
	"context"
	"encoding/json"
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

// same payload as the old pingJob
func TestKeepalivePing(t *testing.T) {
	w := &wsstest.Writer{}
	require.NoError(t, ping(context.Background(), w))
	assert.Equal(t, []string{`{"id":1,"type":"ping"}`}, w.Frames())
}

// tokenServer is a fake bullet API; the i-th call returns intervals[i] (the
// last one repeats). A negative value omits instanceServers.
func tokenServer(t *testing.T, intervals ...int) *httptest.Server {
	t.Helper()
	var calls atomic.Int64
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := int(calls.Add(1) - 1)
		if i >= len(intervals) {
			i = len(intervals) - 1
		}
		servers := "[]"
		if intervals[i] >= 0 {
			servers = fmt.Sprintf(`[{"pingInterval":%d}]`, intervals[i])
		}
		_, _ = fmt.Fprintf(w, `{"data":{"token":"tok","instanceServers":%s}}`, servers)
	}))
	t.Cleanup(s.Close)
	return s
}

// pingServer counts kucoin pings per accepted connection.
type pingServer struct {
	*httptest.Server
	mu    sync.Mutex
	pings []int
}

func newPingServer(t *testing.T) *pingServer {
	t.Helper()
	ps := &pingServer{}
	ps.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		ps.mu.Lock()
		idx := len(ps.pings)
		ps.pings = append(ps.pings, 0)
		ps.mu.Unlock()
		for {
			_, data, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			var msg Ping
			if json.Unmarshal(data, &msg) == nil && msg.Type == "ping" {
				ps.mu.Lock()
				ps.pings[idx]++
				ps.mu.Unlock()
			}
		}
	}))
	t.Cleanup(ps.Server.Close)
	return ps
}

func (ps *pingServer) url() string { return "ws" + ps.URL[len("http"):] }

func (ps *pingServer) snapshot() []int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return append([]int(nil), ps.pings...)
}

func (ps *pingServer) waitConns(n int, timeout time.Duration) []int {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if s := ps.snapshot(); len(s) >= n {
			return s
		}
		time.Sleep(10 * time.Millisecond)
	}
	return ps.snapshot()
}

func runHelper(t *testing.T, ws *wss.WebsocketHelper) func() {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		ws.Run(ctx, func(context.Context, map[string]any) error { return nil })
		close(done)
	}()
	stop := func() {
		cancel()
		<-done
	}
	t.Cleanup(stop)
	return stop
}

// #2640: the bullet-API interval is applied to the connection it was fetched
// for; 0 and a missing instance server fall back to the default (18s, so no
// ping within the ~2s connection: 1s post-dial wait + 1s reconnect interval;
// keepalive starts at dial) instead of panicking.
func TestBulletPingIntervalPerConnection(t *testing.T) {
	tokens := tokenServer(t, 50, 0, 400, -1)
	ps := newPingServer(t)
	ws, err := newWebsocketHelper(context.Background(), tokens.URL,
		wss.WithEndpoint(ps.url()),
		wss.WithReconnectInterval(time.Second))
	require.NoError(t, err)
	stop := runHelper(t, ws)

	conns := ps.waitConns(5, 15*time.Second)
	stop()
	require.GreaterOrEqual(t, len(conns), 5)
	assert.GreaterOrEqual(t, conns[0], 10, "50ms interval")
	assert.Equal(t, 0, conns[1], "0 must fall back to the 18s default")
	assert.GreaterOrEqual(t, conns[2], 3, "400ms interval")
	assert.LessOrEqual(t, conns[2], 6, "400ms interval")
	assert.Equal(t, 0, conns[3], "missing instance server must fall back to the default")
}

// two fetchers with different bullet intervals do not share state (-race)
func TestBulletPingIntervalNotShared(t *testing.T) {
	fastTokens := tokenServer(t, 50)
	slowTokens := tokenServer(t, 0)
	fast, slow := newPingServer(t), newPingServer(t)

	fastWs, err := newWebsocketHelper(context.Background(), fastTokens.URL, wss.WithEndpoint(fast.url()), wss.WithReconnectInterval(time.Second))
	require.NoError(t, err)
	slowWs, err := newWebsocketHelper(context.Background(), slowTokens.URL, wss.WithEndpoint(slow.url()), wss.WithReconnectInterval(time.Second))
	require.NoError(t, err)
	stopFast := runHelper(t, fastWs)
	stopSlow := runHelper(t, slowWs)

	fastConns := fast.waitConns(3, 10*time.Second)
	slowConns := slow.waitConns(3, 10*time.Second)
	stopFast()
	stopSlow()
	require.GreaterOrEqual(t, len(fastConns), 3)
	require.GreaterOrEqual(t, len(slowConns), 3)
	for i := 0; i < 2; i++ {
		assert.GreaterOrEqual(t, fastConns[i], 10, "fast conn %d", i)
		assert.Equal(t, 0, slowConns[i], "slow conn %d", i)
	}
}
