package hub

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/dal/collector"
	dalcommon "bisonai.com/miko/node/pkg/dal/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// wsServer registers every connection and handles its SUBSCRIBE messages.
func wsServer(t *testing.T, h *Hub) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		h.Register <- c
		defer func() { h.Unregister <- c }()
		for {
			var msg Subscription
			if err := wsjson.Read(r.Context(), c, &msg); err != nil {
				return
			}
			h.HandleSubscription(c, msg)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func subscriptionsOf(h *Hub, symbol string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	n := 0
	for _, subs := range h.Clients {
		if _, ok := subs[symbol]; ok {
			n++
		}
	}
	return n
}

func TestAddRemoveSymbols(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	oldStream := make(chan *dalcommon.OutgoingSubmissionData, 10)
	col := &collector.Collector{OutgoingStream: map[string]chan *dalcommon.OutgoingSubmissionData{"A-B": oldStream}}
	h := NewHub(map[string]struct{}{"A-B": {}})
	h.Start(ctx, col)

	newStream := make(chan *dalcommon.OutgoingSubmissionData, 10)
	h.AddSymbols(map[string]chan *dalcommon.OutgoingSubmissionData{"C-D": newStream})
	assert.ElementsMatch(t, []string{"A-B", "C-D"}, h.SymbolList())

	srv := wsServer(t, h)
	conn, _, err := websocket.Dial(ctx, "ws"+srv.URL[len("http"):], nil)
	require.NoError(t, err)
	defer conn.Close(websocket.StatusNormalClosure, "")

	require.NoError(t, wsjson.Write(ctx, conn, Subscription{Method: "SUBSCRIBE", Params: []string{"submission@A-B", "submission@C-D"}}))
	require.Eventually(t, func() bool { return subscriptionsOf(h, "C-D") == 1 }, 2*time.Second, 10*time.Millisecond)

	// added symbol is broadcast
	newStream <- &dalcommon.OutgoingSubmissionData{Symbol: "C-D", Value: "1"}
	readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second)
	defer readCancel()
	var got dalcommon.OutgoingSubmissionData
	require.NoError(t, wsjson.Read(readCtx, conn, &got))
	assert.Equal(t, "C-D", got.Symbol)

	// removed symbol: unsubscribable, dropped from client subscriptions
	h.RemoveSymbols([]string{"A-B"})
	assert.Equal(t, []string{"C-D"}, h.SymbolList())
	assert.Equal(t, 0, subscriptionsOf(h, "A-B"))
	assert.Equal(t, 1, subscriptionsOf(h, "C-D"))
	h.mu.RLock()
	var client *websocket.Conn
	for c := range h.Clients {
		client = c
	}
	h.mu.RUnlock()
	assert.Equal(t, 0, h.HandleSubscription(client, Subscription{Method: "SUBSCRIBE", Params: []string{"submission@A-B"}}))

	// data still in flight for the removed symbol is not sent
	oldStream <- &dalcommon.OutgoingSubmissionData{Symbol: "A-B", Value: "2"}
	close(oldStream)
	newStream <- &dalcommon.OutgoingSubmissionData{Symbol: "C-D", Value: "3"}
	require.NoError(t, wsjson.Read(readCtx, conn, &got))
	assert.Equal(t, "C-D", got.Symbol)
	assert.Equal(t, "3", got.Value)
}

func TestAddSymbolsBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream := make(chan *dalcommon.OutgoingSubmissionData, 1)
	col := &collector.Collector{OutgoingStream: map[string]chan *dalcommon.OutgoingSubmissionData{"C-D": stream}}
	h := NewHub(map[string]struct{}{})
	h.AddSymbols(map[string]chan *dalcommon.OutgoingSubmissionData{"C-D": stream})
	h.Start(ctx, col)

	// one broadcaster drains the stream
	stream <- &dalcommon.OutgoingSubmissionData{Symbol: "C-D"}
	require.Eventually(t, func() bool { return len(stream) == 0 }, 2*time.Second, 10*time.Millisecond)
}
