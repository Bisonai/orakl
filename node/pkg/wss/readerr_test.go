package wss

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"nhooyr.io/websocket"
)

func TestReadErrorLevel(t *testing.T) {
	live := context.Background()
	assert.Equal(t, zerolog.ErrorLevel, ReadErrorLevel(live, zerolog.ErrorLevel))
	assert.Equal(t, zerolog.WarnLevel, ReadErrorLevel(live, zerolog.WarnLevel))

	cancelled, cancel := context.WithCancel(live)
	cancel()
	assert.Equal(t, zerolog.DebugLevel, ReadErrorLevel(cancelled, zerolog.ErrorLevel))
}

// Stopping Run fails the blocked read with its ctx already done, so a custom
// read func logs it at debug, not error (#2985).
func TestStopFailsReadWithDoneCtx(t *testing.T) {
	rs := newRecordingServer(t, nil)
	type result struct {
		err   error
		level zerolog.Level
	}
	results := make(chan result, 10)
	readFunc := func(ctx context.Context, conn *websocket.Conn) (map[string]interface{}, error) {
		_, _, err := conn.Read(ctx)
		if err != nil {
			results <- result{err, ReadErrorLevel(ctx, zerolog.ErrorLevel)}
		}
		return nil, err
	}
	ws, err := NewWebsocketHelper(context.Background(), WithEndpoint(rs.url()), WithCustomReadFunc(readFunc))
	if err != nil {
		t.Fatal(err)
	}
	ws.subscribeDelay = 10 * time.Millisecond
	stop := startRun(t, ws, nil)
	if len(rs.waitDials(1, 5*time.Second)) < 1 {
		t.Fatal("no dial")
	}
	// let the read block
	time.Sleep(100 * time.Millisecond)
	stop()

	select {
	case r := <-results:
		assert.Error(t, r.err)
		assert.Equal(t, zerolog.DebugLevel, r.level, "stop read error: %v", r.err)
	case <-time.After(3 * time.Second):
		t.Fatal("read did not fail on stop")
	}
}

// A peer dropping the connection fails the read with its ctx live: that one
// keeps its level.
func TestPeerCloseFailsReadWithLiveCtx(t *testing.T) {
	rs := newRecordingServer(t, func(ctx context.Context, conn *websocket.Conn, idx int) {
		if idx != 0 {
			<-ctx.Done()
			return
		}
		if sleepCtx(ctx, 200*time.Millisecond) != nil {
			return
		}
		_ = conn.CloseNow()
	})
	type result struct {
		err   error
		level zerolog.Level
	}
	results := make(chan result, 10)
	readFunc := func(ctx context.Context, conn *websocket.Conn) (map[string]interface{}, error) {
		_, _, err := conn.Read(ctx)
		if err != nil {
			results <- result{err, ReadErrorLevel(ctx, zerolog.ErrorLevel)}
		}
		return nil, err
	}
	ws, err := NewWebsocketHelper(context.Background(), WithEndpoint(rs.url()), WithCustomReadFunc(readFunc))
	if err != nil {
		t.Fatal(err)
	}
	ws.subscribeDelay = 10 * time.Millisecond
	stop := startRun(t, ws, nil)
	defer stop()

	select {
	case r := <-results:
		assert.False(t, errors.Is(r.err, context.Canceled), "unexpected ctx error: %v", r.err)
		assert.Equal(t, zerolog.ErrorLevel, r.level, "peer close read error: %v", r.err)
	case <-time.After(5 * time.Second):
		t.Fatal("read did not fail on peer close")
	}
}
