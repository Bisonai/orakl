package hashkey

import (
	"context"
	"encoding/json"
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

// same payload as the old pingJob: {"ping":<unix ms>}
func TestKeepalivePing(t *testing.T) {
	w := &wsstest.Writer{}
	before := time.Now().UnixMilli()
	require.NoError(t, ping(context.Background(), w))
	after := time.Now().UnixMilli()

	frames := w.Frames()
	require.Len(t, frames, 1)
	var got map[string]int64
	require.NoError(t, json.Unmarshal([]byte(frames[0]), &got))
	require.Len(t, got, 1)
	assert.GreaterOrEqual(t, got["ping"], before)
	assert.LessOrEqual(t, got["ping"], after)
	assert.Equal(t, 10000, PingInterval)
}

// #2625: the keepalive must work after a failed initial dial and resume on
// every later connection.
func TestKeepaliveSurvivesFailedInitialDial(t *testing.T) {
	var handshakes atomic.Int64
	var mu sync.Mutex
	pings := map[int64]int{} // conn index -> pings received
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idx := handshakes.Add(1) - 1
		if idx == 0 {
			http.Error(w, "nope", http.StatusMethodNotAllowed)
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		for {
			_, data, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			var msg Ping
			if json.Unmarshal(data, &msg) == nil && msg.Ping > 0 {
				mu.Lock()
				pings[idx]++
				n := pings[idx]
				mu.Unlock()
				// drop the first good conn after a few pings to force a reconnect
				if idx == 1 && n == 3 {
					_ = conn.Close(websocket.StatusNormalClosure, "")
					return
				}
			}
		}
	}))
	defer server.Close()

	ws, err := wss.NewWebsocketHelper(context.Background(),
		wss.WithEndpoint("ws"+server.URL[len("http"):]),
		wss.WithKeepalive(50*time.Millisecond, ping),
	)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		ws.Run(ctx, func(context.Context, map[string]any) error { return nil })
		close(done)
	}()
	defer func() {
		cancel()
		<-done
	}()

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return pings[1] >= 3 && pings[2] >= 3
	}, 10*time.Second, 20*time.Millisecond, "keepalive did not run on the connections after a failed dial")
	assert.GreaterOrEqual(t, handshakes.Load(), int64(3))
}
