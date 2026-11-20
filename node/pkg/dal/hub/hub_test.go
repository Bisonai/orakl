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

	// removed symbol: unsubscribable, client subscriptions kept for a re-add
	h.RemoveSymbols([]string{"A-B"})
	assert.Equal(t, []string{"C-D"}, h.SymbolList())
	assert.Equal(t, 1, subscriptionsOf(h, "A-B"))
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

// A client whose close handshake blocks (peer never reads, so never answers
// the close frame) must not stall a concurrent broadcast to another client.
func TestSlowCloseDoesNotStallBroadcast(t *testing.T) {
	cases := map[string]func(h *Hub, slow *websocket.Conn){
		"removeClient": func(h *Hub, slow *websocket.Conn) { h.removeClient(slow) },
		"cleanup":      func(h *Hub, _ *websocket.Conn) { h.cleanup() },
	}
	for name, remove := range cases {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			stream := make(chan *dalcommon.OutgoingSubmissionData)
			col := &collector.Collector{OutgoingStream: map[string]chan *dalcommon.OutgoingSubmissionData{"A-B": stream}}
			h := NewHub(map[string]struct{}{"A-B": {}})
			h.Start(ctx, col)
			srv := wsServer(t, h)
			url := "ws" + srv.URL[len("http"):]

			healthy, _, err := websocket.Dial(ctx, url, nil)
			require.NoError(t, err)
			defer healthy.CloseNow()
			require.NoError(t, wsjson.Write(ctx, healthy, Subscription{Method: "SUBSCRIBE", Params: []string{"submission@A-B"}}))
			require.Eventually(t, func() bool { return subscriptionsOf(h, "A-B") == 1 }, 2*time.Second, 10*time.Millisecond)

			// slow peer: never reads, never subscribes
			slowPeer, _, err := websocket.Dial(ctx, url, nil)
			require.NoError(t, err)
			defer slowPeer.CloseNow()

			var slow *websocket.Conn
			require.Eventually(t, func() bool {
				h.mu.RLock()
				defer h.mu.RUnlock()
				for c, subs := range h.Clients {
					if len(subs) == 0 {
						slow = c
						return true
					}
				}
				return false
			}, 2*time.Second, 10*time.Millisecond)

			const bound = 2 * time.Second // close handshake waits ~5 s
			start := time.Now()
			go remove(h, slow)

			removed := make(chan struct{})
			go func() {
				defer close(removed)
				for {
					h.mu.RLock()
					_, ok := h.Clients[slow]
					h.mu.RUnlock()
					if !ok {
						return
					}
					time.Sleep(5 * time.Millisecond)
				}
			}()
			select {
			case <-removed:
			case <-time.After(bound):
				t.Fatal("hub lock held while closing slow client")
			}

			broadcastDone := make(chan struct{})
			go func() {
				defer close(broadcastDone)
				h.castSubmissionData(ctx, &dalcommon.OutgoingSubmissionData{Symbol: "A-B", Value: "1"}, "A-B", stream)
			}()

			readCtx, readCancel := context.WithTimeout(ctx, bound)
			defer readCancel()
			var got dalcommon.OutgoingSubmissionData
			require.NoError(t, wsjson.Read(readCtx, healthy, &got))
			assert.Equal(t, "1", got.Value)

			select {
			case <-broadcastDone:
			case <-time.After(bound):
				t.Fatal("broadcast stalled by slow client close")
			}
			assert.Less(t, time.Since(start), bound)
		})
	}
}

// A client subscribed to a symbol gets nothing for it while a reload has
// removed it, and receives it again once a later reload re-adds it, without
// resubscribing.
func TestRemovedThenReAddedSymbolKeepsSubscription(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	oldStream := make(chan *dalcommon.OutgoingSubmissionData, 10)
	col := &collector.Collector{OutgoingStream: map[string]chan *dalcommon.OutgoingSubmissionData{"A-B": oldStream}}
	h := NewHub(map[string]struct{}{"A-B": {}})
	h.Start(ctx, col)

	srv := wsServer(t, h)
	conn, _, err := websocket.Dial(ctx, "ws"+srv.URL[len("http"):], nil)
	require.NoError(t, err)
	defer conn.CloseNow()
	require.NoError(t, wsjson.Write(ctx, conn, Subscription{Method: "SUBSCRIBE", Params: []string{"submission@A-B"}}))
	require.Eventually(t, func() bool { return subscriptionsOf(h, "A-B") == 1 }, 2*time.Second, 10*time.Millisecond)

	// Reads run in one goroutine: a websocket read must not be abandoned midway.
	msgs := make(chan dalcommon.OutgoingSubmissionData, 10)
	go func() {
		for {
			var got dalcommon.OutgoingSubmissionData
			if err := wsjson.Read(ctx, conn, &got); err != nil {
				return
			}
			msgs <- got
		}
	}()
	receive := func(timeout time.Duration) (dalcommon.OutgoingSubmissionData, bool) {
		select {
		case got := <-msgs:
			return got, true
		case <-time.After(timeout):
			return dalcommon.OutgoingSubmissionData{}, false
		}
	}

	oldStream <- &dalcommon.OutgoingSubmissionData{Symbol: "A-B", Value: "1"}
	got, ok := receive(2 * time.Second)
	require.True(t, ok)
	assert.Equal(t, "1", got.Value)

	// reload removes A-B (hub, then collector closes the stream)
	h.RemoveSymbols([]string{"A-B"})
	oldStream <- &dalcommon.OutgoingSubmissionData{Symbol: "A-B", Value: "2"}
	close(oldStream)
	got, ok = receive(300 * time.Millisecond)
	assert.False(t, ok, "removed symbol delivered: %+v", got)

	// reload re-adds A-B with a new stream
	newStream := make(chan *dalcommon.OutgoingSubmissionData, 10)
	h.AddSymbols(map[string]chan *dalcommon.OutgoingSubmissionData{"A-B": newStream})
	newStream <- &dalcommon.OutgoingSubmissionData{Symbol: "A-B", Value: "3"}
	got, ok = receive(2 * time.Second)
	require.True(t, ok, "re-added symbol not delivered to existing subscriber")
	assert.Equal(t, "3", got.Value)

	// data from the old stream (still draining after a quick re-add) is not delivered
	h.castSubmissionData(ctx, &dalcommon.OutgoingSubmissionData{Symbol: "A-B", Value: "stale"}, "A-B", oldStream)
	newStream <- &dalcommon.OutgoingSubmissionData{Symbol: "A-B", Value: "4"}
	got, ok = receive(2 * time.Second)
	require.True(t, ok)
	assert.Equal(t, "4", got.Value)
}
