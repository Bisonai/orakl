//nolint:all
package wss

import (
	"context"
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

func TestWriteWithoutConnReturnsError(t *testing.T) {
	rs := newRecordingServer(t, nil)
	ws, err := NewWebsocketHelper(context.Background(), WithEndpoint(rs.url()))
	require.NoError(t, err)
	ctx := context.Background()

	// before the first dial
	assert.ErrorIs(t, ws.Write(ctx, map[string]any{"a": 1}), ErrNotConnected)
	assert.ErrorIs(t, ws.RawWrite(ctx, "x"), ErrNotConnected)
	assert.ErrorIs(t, ws.IsAlive(ctx), ErrNotConnected)
	assert.NoError(t, ws.Close())

	require.NoError(t, ws.Dial(ctx))
	assert.NoError(t, ws.Write(ctx, map[string]any{"a": 1}))
	assert.NoError(t, ws.Close())

	// after Close; Close stays idempotent
	assert.ErrorIs(t, ws.Write(ctx, map[string]any{"a": 1}), ErrNotConnected)
	assert.ErrorIs(t, ws.RawWrite(ctx, "x"), ErrNotConnected)
	assert.NoError(t, ws.Close())
}

func TestConcurrentWritesAcrossReconnects(t *testing.T) {
	rs := newRecordingServer(t, nil)
	// every cycle = dial + 1s pre-subscribe wait + 100ms
	ws, err := NewWebsocketHelper(context.Background(), WithEndpoint(rs.url()), WithReconnectInterval(100*time.Millisecond))
	require.NoError(t, err)
	stop := startRun(t, ws, nil)

	writeCtx, cancelWrites := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	var ok atomic.Int64
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for writeCtx.Err() == nil {
				var err error
				if i%2 == 0 {
					err = ws.Write(writeCtx, map[string]any{"w": i})
				} else {
					err = ws.RawWrite(writeCtx, "raw")
				}
				// errors are expected while reconnecting
				if err == nil {
					ok.Add(1)
				}
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	conns := rs.waitDials(3, 6*time.Second)
	cancelWrites()
	wg.Wait()
	stop()
	assert.GreaterOrEqual(t, len(conns), 3, "expected forced reconnects")
	assert.Positive(t, ok.Load(), "no write succeeded")
}

func TestRouterCtxCancelledWhenConnEnds(t *testing.T) {
	rs := newRecordingServer(t, func(ctx context.Context, conn *websocket.Conn, idx int) {
		if idx != 0 {
			return
		}
		// client reads ~1s after dial; send one message, then drop the conn
		if sleepCtx(ctx, 1300*time.Millisecond) != nil {
			return
		}
		_ = wsjson.Write(ctx, conn, map[string]any{"a": 1})
		if sleepCtx(ctx, 200*time.Millisecond) != nil {
			return
		}
		_ = conn.Close(websocket.StatusNormalClosure, "")
	})

	routerCtxs := make(chan context.Context, 1)
	ws, err := NewWebsocketHelper(context.Background(), WithEndpoint(rs.url()))
	require.NoError(t, err)
	startRun(t, ws, func(ctx context.Context, _ map[string]any) error {
		routerCtxs <- ctx
		return nil
	})

	var routerCtx context.Context
	select {
	case routerCtx = <-routerCtxs:
	case <-time.After(5 * time.Second):
		t.Fatal("router was not called")
	}

	select {
	case <-routerCtx.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("router ctx not cancelled after its connection ended")
	}
	// only the connection ended, not Run
	assert.True(t, ws.IsRunning())
}

func TestWriterFromContextBoundToItsConn(t *testing.T) {
	rs := newRecordingServer(t, func(ctx context.Context, conn *websocket.Conn, idx int) {
		if idx != 0 {
			return
		}
		if sleepCtx(ctx, 1300*time.Millisecond) != nil {
			return
		}
		_ = wsjson.Write(ctx, conn, map[string]any{"from": 0})
		// close only after the router reply had time to arrive
		if sleepCtx(ctx, 300*time.Millisecond) != nil {
			return
		}
		_ = conn.Close(websocket.StatusNormalClosure, "")
	})

	_, ok := WriterFromContext(context.Background())
	assert.False(t, ok, "plain ctx must not carry a writer")

	writers := make(chan ConnWriter, 1)
	replyErrs := make(chan error, 1)
	ws, err := NewWebsocketHelper(context.Background(), WithEndpoint(rs.url()))
	require.NoError(t, err)
	startRun(t, ws, func(ctx context.Context, _ map[string]any) error {
		w, ok := WriterFromContext(ctx)
		if !ok {
			replyErrs <- assert.AnError
			return nil
		}
		replyErrs <- w.RawWrite(ctx, "reply")
		writers <- w
		return nil
	})

	var stale ConnWriter
	select {
	case err := <-replyErrs:
		require.NoError(t, err, "reply via conn-bound writer failed")
		stale = <-writers
	case <-time.After(5 * time.Second):
		t.Fatal("router was not called")
	}

	// wait until the replacement connection is live on the client side
	require.GreaterOrEqual(t, len(rs.waitDials(2, 5*time.Second)), 2, "no reconnect")
	require.Eventually(t, func() bool {
		return ws.RawWrite(context.Background(), "probe") == nil
	}, 3*time.Second, 10*time.Millisecond)

	assert.ErrorIs(t, stale.RawWrite(context.Background(), "stale"), ErrConnClosed)
	assert.ErrorIs(t, stale.Write(context.Background(), map[string]any{"stale": true}), ErrConnClosed)

	// a marker after the stale writes; once it arrives, anything sent before it
	// on the same conn has arrived too
	require.NoError(t, ws.RawWrite(context.Background(), "marker"))
	require.Eventually(t, func() bool {
		s := rs.snapshot()
		return len(s) >= 2 && contains(s[1].msgs, "marker")
	}, 3*time.Second, 10*time.Millisecond)

	s := rs.snapshot()
	assert.Contains(t, s[0].msgs, "reply", "reply did not reach the conn the message came on")
	for _, m := range s[1].msgs {
		assert.NotContains(t, m, "stale", "stale write reached the new connection")
	}
}

func TestRunConcurrentCallsOnlyOneRuns(t *testing.T) {
	rs := newRecordingServer(t, nil)
	ws, err := NewWebsocketHelper(context.Background(), WithEndpoint(rs.url()))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	start := make(chan struct{})
	returned := make(chan struct{}, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			ws.Run(ctx, func(context.Context, map[string]any) error { return nil })
			returned <- struct{}{}
		}()
	}
	close(start)

	// the loser returns at once, the winner keeps running
	select {
	case <-returned:
	case <-time.After(2 * time.Second):
		t.Fatal("second Run did not return")
	}
	time.Sleep(1500 * time.Millisecond)
	assert.Len(t, returned, 0, "both Run calls returned")
	assert.True(t, ws.IsRunning())
	assert.Len(t, rs.snapshot(), 1, "more than one Run dialed")

	cancel()
	select {
	case <-returned:
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
	assert.False(t, ws.IsRunning())
}

func contains(msgs []string, want string) bool {
	for _, m := range msgs {
		if strings.Contains(m, want) {
			return true
		}
	}
	return false
}

func TestWriteWithDoneCtxKeepsConn(t *testing.T) {
	rs := newRecordingServer(t, nil)
	ws, err := NewWebsocketHelper(context.Background(), WithEndpoint(rs.url()))
	require.NoError(t, err)
	require.NoError(t, ws.Dial(context.Background()))
	defer ws.Close()

	done, cancel := context.WithCancel(context.Background())
	cancel()
	// nhooyr closes the conn when a write ctx expires; a done ctx (e.g. from a
	// previous connection's router) must not reach it
	for i := 0; i < 50; i++ {
		assert.ErrorIs(t, ws.Write(done, map[string]any{"a": 1}), context.Canceled)
		assert.ErrorIs(t, ws.RawWrite(done, "x"), context.Canceled)
	}
	time.Sleep(50 * time.Millisecond)
	assert.NoError(t, ws.RawWrite(context.Background(), "still-open"))
}
