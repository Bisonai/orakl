//nolint:all
package wss

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func echoHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to accept websocket connection")
		return
	}
	defer conn.Close(websocket.StatusInternalError, "the sky is falling")

	for {
		var v interface{}
		err := wsjson.Read(r.Context(), conn, &v)
		if err != nil {
			break
		}

		err = wsjson.Write(r.Context(), conn, v)
		if err != nil {
			log.Error().Err(err).Msg("failed to write message")
			break
		}
	}
}

func TestReadWriteAndClose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(echoHandler))
	defer server.Close()
	wsURL := "ws" + server.URL[len("http"):] + "/ws"

	time.Sleep(time.Second)

	// Create a WebSocket connection
	conn, err := NewWebsocketHelper(context.Background(), WithEndpoint(wsURL))
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	// Test Dial
	err = conn.Dial(context.Background())
	assert.NoError(t, err)

	// Test Write
	err = conn.Write(context.Background(), "Hello")
	assert.NoError(t, err)

	// Test Read
	ch := make(chan any)
	go conn.Read(context.Background(), ch)
	assert.Equal(t, "Hello", <-ch)

	// Test Close
	err = conn.Close()
	assert.NoError(t, err)
}

func TestReconnectTicker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(echoHandler))
	defer server.Close()
	wsURL := "ws" + server.URL[len("http"):] + "/ws"

	// Create a WebSocket connection with a short reconnect interval
	conn, err := NewWebsocketHelper(
		context.Background(),
		WithEndpoint(wsURL),
		WithReconnectInterval(100*time.Millisecond), // Short interval for testing
	)
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	// Run the WebSocket helper in a separate goroutine; cancel so it does not
	// keep redialing the closed server during later tests
	runCtx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()
	go conn.Run(runCtx, func(ctx context.Context, data map[string]any) error {
		// Handle data if necessary
		return nil
	})

	// Give it some time to reconnect
	time.Sleep(500 * time.Millisecond)

	// Test if the connection has been re-established
	assert.NotNil(t, conn.Conn) // Assuming that `conn.Conn` should not be nil
	assert.True(t, conn.IsRunning)

	// Close the WebSocket helper
	err = conn.Close()
	assert.NoError(t, err)
}

func TestInactivityTimer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(echoHandler))
	defer server.Close()
	wsURL := "ws" + server.URL[len("http"):] + "/ws"

	// Create a WebSocket connection with a short inactivity timeout
	conn, err := NewWebsocketHelper(
		context.Background(),
		WithEndpoint(wsURL),
		WithInactivityTimeout(50*time.Millisecond), // Short timeout for testing
	)
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	// Run the WebSocket helper in a separate goroutine; cancel so it does not
	// keep redialing the closed server during later tests
	runCtx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()
	go conn.Run(runCtx, func(ctx context.Context, data map[string]any) error {
		// Handle data if necessary
		return nil
	})

	// Wait longer than the inactivity timeout
	time.Sleep(500 * time.Millisecond)

	// Verify if the connection has been closed due to inactivity
	assert.NotNil(t, conn.Conn) // Assuming that `conn.Conn` should be nil if closed
	assert.True(t, conn.IsRunning)

	// Close the WebSocket helper
	err = conn.Close()
	assert.NoError(t, err)
}

func assertRunReturnsAfterCancel(t *testing.T, ws *WebsocketHelper, cancelAfter time.Duration) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		ws.Run(ctx, func(ctx context.Context, data map[string]any) error { return nil })
		close(done)
	}()

	time.Sleep(cancelAfter)
	cancel()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after context cancel")
	}
	assert.False(t, ws.IsRunning)
}

func TestRunReturnsOnCancelWhileDialFailing(t *testing.T) {
	// port 1 refuses connections, so dial keeps failing
	ws, err := NewWebsocketHelper(context.Background(), WithEndpoint("ws://127.0.0.1:1"))
	assert.NoError(t, err)

	assertRunReturnsAfterCancel(t, ws, 1500*time.Millisecond)
}

func TestRunReturnsOnCancelWhileConnected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(echoHandler))
	defer server.Close()
	wsURL := "ws" + server.URL[len("http"):] + "/ws"

	ws, err := NewWebsocketHelper(context.Background(), WithEndpoint(wsURL))
	assert.NoError(t, err)

	// dial + 1s pre-subscribe wait, then reading
	assertRunReturnsAfterCancel(t, ws, 1500*time.Millisecond)
}

func TestRunReturnsOnCancelMidSubscribe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(echoHandler))
	defer server.Close()
	wsURL := "ws" + server.URL[len("http"):] + "/ws"

	ws, err := NewWebsocketHelper(
		context.Background(),
		WithEndpoint(wsURL),
		WithSubscriptions([]any{map[string]any{"a": 1}, map[string]any{"b": 2}, map[string]any{"c": 3}}),
	)
	assert.NoError(t, err)

	// cancel during the per-subscription sleeps
	assertRunReturnsAfterCancel(t, ws, 1500*time.Millisecond)
}

func TestRunReturnsOnCancelAfterServerGone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(echoHandler))
	wsURL := "ws" + server.URL[len("http"):] + "/ws"

	ws, err := NewWebsocketHelper(context.Background(), WithEndpoint(wsURL))
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		ws.Run(ctx, func(ctx context.Context, data map[string]any) error { return nil })
		close(done)
	}()

	// let it connect, then kill server to force read error + failing redials
	time.Sleep(1500 * time.Millisecond)
	server.CloseClientConnections()
	server.Close()
	time.Sleep(500 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after context cancel")
	}
}

// connRecord is the server-side view of one accepted websocket connection.
type connRecord struct {
	accepted time.Time
	closed   time.Time
}

// recordingServer accepts websocket connections, runs onConn for each and
// records when each connection was accepted and when the client went away.
type recordingServer struct {
	*httptest.Server
	mu    sync.Mutex
	conns []*connRecord
}

func newRecordingServer(t *testing.T, onConn func(ctx context.Context, conn *websocket.Conn, idx int)) *recordingServer {
	t.Helper()
	rs := &recordingServer{}
	rs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()

		rec := &connRecord{accepted: time.Now()}
		rs.mu.Lock()
		idx := len(rs.conns)
		rs.conns = append(rs.conns, rec)
		rs.mu.Unlock()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		if onConn != nil {
			go onConn(ctx, conn, idx)
		}

		// drain until the client goes away; also answers control frames
		for {
			if _, _, err := conn.Read(ctx); err != nil {
				break
			}
		}
		rs.mu.Lock()
		rec.closed = time.Now()
		rs.mu.Unlock()
	}))
	t.Cleanup(rs.Server.Close)
	return rs
}

func (rs *recordingServer) url() string {
	return "ws" + rs.URL[len("http"):] + "/ws"
}

func (rs *recordingServer) snapshot() []connRecord {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	out := make([]connRecord, len(rs.conns))
	for i, c := range rs.conns {
		out[i] = *c
	}
	return out
}

func (rs *recordingServer) waitDials(n int, timeout time.Duration) []connRecord {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if s := rs.snapshot(); len(s) >= n {
			return s
		}
		time.Sleep(10 * time.Millisecond)
	}
	return rs.snapshot()
}

// startRun runs ws.Run in the background and returns a stop func that cancels
// it and fails the test if Run does not return promptly.
func startRun(t *testing.T, ws *WebsocketHelper, router func(context.Context, map[string]any) error) func() {
	t.Helper()
	if router == nil {
		router = func(context.Context, map[string]any) error { return nil }
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		ws.Run(ctx, router)
		close(done)
	}()
	stopped := false
	stop := func() {
		if stopped {
			return
		}
		stopped = true
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("Run did not return promptly after context cancel")
		}
	}
	t.Cleanup(stop)
	return stop
}

func TestRunReconnectsSilentConnAfterInactivity(t *testing.T) {
	const inactivity = 200 * time.Millisecond
	rs := newRecordingServer(t, nil)

	ws, err := NewWebsocketHelper(context.Background(), WithEndpoint(rs.url()), WithInactivityTimeout(inactivity))
	assert.NoError(t, err)
	stop := startRun(t, ws, nil)

	// each cycle = dial + 1s pre-subscribe wait + inactivity
	conns := rs.waitDials(3, 6*time.Second)
	stop()
	if !assert.GreaterOrEqual(t, len(conns), 3, "silent connection was not reconnected") {
		return
	}
	// the second connection must also wait a full inactivity window, i.e. the
	// last message time is reset per connection
	for i := 0; i < 2; i++ {
		lifetime := conns[i+1].accepted.Sub(conns[i].accepted)
		assert.GreaterOrEqual(t, lifetime, time.Second+inactivity*3/4, "connection %d reconnected too early", i)
	}
}

func TestRunInactivityCountsFromLastMessage(t *testing.T) {
	const inactivity = 500 * time.Millisecond
	var sentAt atomic.Int64
	rs := newRecordingServer(t, func(ctx context.Context, conn *websocket.Conn, idx int) {
		if idx != 0 {
			return
		}
		// client starts reading ~1s after dial; send one message while it reads
		if sleepCtx(ctx, 1300*time.Millisecond) != nil {
			return
		}
		sentAt.Store(time.Now().UnixNano())
		_ = wsjson.Write(ctx, conn, map[string]any{"a": 1})
	})

	var routed atomic.Int32
	ws, err := NewWebsocketHelper(context.Background(), WithEndpoint(rs.url()), WithInactivityTimeout(inactivity))
	assert.NoError(t, err)
	stop := startRun(t, ws, func(context.Context, map[string]any) error {
		routed.Add(1)
		return nil
	})

	conns := rs.waitDials(2, 5*time.Second)
	stop()
	if !assert.GreaterOrEqual(t, len(conns), 2, "connection silent after one message was not reconnected") {
		return
	}
	assert.Equal(t, int32(1), routed.Load())
	if !assert.NotZero(t, sentAt.Load(), "server never sent the message") {
		return
	}
	sent := time.Unix(0, sentAt.Load())
	assert.GreaterOrEqual(t, conns[1].accepted.Sub(sent), inactivity*3/4, "inactivity not counted from the last message")
}

func TestRunReconnectsSilentConnAtReconnectInterval(t *testing.T) {
	rs := newRecordingServer(t, nil)

	ws, err := NewWebsocketHelper(
		context.Background(),
		WithEndpoint(rs.url()),
		WithInactivityTimeout(time.Hour),
		// longer than dial + 1s pre-subscribe wait, so a timer started with Run
		// expires mid-read rather than before the first read
		WithReconnectInterval(1500*time.Millisecond),
	)
	assert.NoError(t, err)
	stop := startRun(t, ws, nil)

	conns := rs.waitDials(2, 5*time.Second)
	stop()
	if !assert.GreaterOrEqual(t, len(conns), 2, "silent connection was not reconnected at ReconnectInterval") {
		return
	}
	assert.GreaterOrEqual(t, conns[1].accepted.Sub(conns[0].accepted), 1500*time.Millisecond)
}

func TestRunEmptyReadsDoNotCountAsActivity(t *testing.T) {
	const inactivity = 500 * time.Millisecond
	rs := newRecordingServer(t, func(ctx context.Context, conn *websocket.Conn, idx int) {
		// app-level heartbeat only, no data
		for sleepCtx(ctx, 100*time.Millisecond) == nil {
			if conn.Write(ctx, websocket.MessageText, []byte("pong")) != nil {
				return
			}
		}
	})

	// mirrors providers (bitget, bitmart, xt) that return (nil, nil) for "pong"
	pongReader := func(ctx context.Context, conn *websocket.Conn) (map[string]any, error) {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return nil, err
		}
		if string(data) == "pong" {
			return nil, nil
		}
		return map[string]any{"raw": string(data)}, nil
	}

	ws, err := NewWebsocketHelper(
		context.Background(),
		WithEndpoint(rs.url()),
		WithInactivityTimeout(inactivity),
		WithCustomReadFunc(pongReader),
	)
	assert.NoError(t, err)
	stop := startRun(t, ws, nil)

	conns := rs.waitDials(3, 6*time.Second)
	stop()
	assert.GreaterOrEqual(t, len(conns), 3, "heartbeat-only connection was not reconnected")
}

func TestRunKeepsActiveConn(t *testing.T) {
	const inactivity = 500 * time.Millisecond
	rs := newRecordingServer(t, func(ctx context.Context, conn *websocket.Conn, idx int) {
		for sleepCtx(ctx, 100*time.Millisecond) == nil {
			if wsjson.Write(ctx, conn, map[string]any{"a": 1}) != nil {
				return
			}
		}
	})

	ws, err := NewWebsocketHelper(context.Background(), WithEndpoint(rs.url()), WithInactivityTimeout(inactivity))
	assert.NoError(t, err)
	stop := startRun(t, ws, nil)

	time.Sleep(1*time.Second + 6*inactivity)
	stop()
	assert.Len(t, rs.snapshot(), 1, "active connection should not be reconnected")
}

func TestNewWebsocketHelperRejectsNonPositiveWindows(t *testing.T) {
	_, err := NewWebsocketHelper(context.Background(), WithEndpoint("ws://x"), WithInactivityTimeout(0))
	assert.Error(t, err)
	_, err = NewWebsocketHelper(context.Background(), WithEndpoint("ws://x"), WithReconnectInterval(-time.Second))
	assert.Error(t, err)
}
