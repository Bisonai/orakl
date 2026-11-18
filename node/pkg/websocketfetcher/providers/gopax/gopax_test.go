package gopax

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/wss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
)

// The reader answers primus pings on the conn it read them from (writer from
// the read ctx), with the same bytes as before (f.Ws.RawWrite).
func TestPrimusPingReply(t *testing.T) {
	const ping = `"primus::ping::1763424000123"`
	const want = `"primus::pong::1763424000123"`
	replies := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		// the client starts reading ~1s after dial (pre-subscribe wait)
		time.Sleep(1200 * time.Millisecond)
		if conn.Write(r.Context(), websocket.MessageText, []byte(ping)) != nil {
			return
		}
		_, data, err := conn.Read(r.Context())
		if err != nil {
			return
		}
		replies <- string(data)
		_, _, _ = conn.Read(r.Context())
	}))
	defer server.Close()

	f := &GopaxFetcher{}
	ws, err := wss.NewWebsocketHelper(context.Background(),
		wss.WithEndpoint("ws"+server.URL[len("http"):]),
		wss.WithCustomReadFunc(f.customReadFunc))
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

	select {
	case got := <-replies:
		assert.Equal(t, want, got)
	case <-time.After(5 * time.Second):
		t.Fatal("no pong received")
	}
}
