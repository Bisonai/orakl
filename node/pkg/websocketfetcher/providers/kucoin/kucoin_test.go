package kucoin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/wss"
	"nhooyr.io/websocket"
)

func pingCounterServer(count *atomic.Int64) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		for {
			_, data, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			var msg Ping
			if json.Unmarshal(data, &msg) == nil && msg.Type == "ping" {
				count.Add(1)
			}
		}
	}))
}

func TestPingJobStopsOnContextCancel(t *testing.T) {
	var pings atomic.Int64
	server := pingCounterServer(&pings)
	defer server.Close()

	ws, err := wss.NewWebsocketHelper(context.Background(), wss.WithEndpoint("ws"+server.URL[len("http"):]))
	if err != nil {
		t.Fatal(err)
	}
	if err = ws.Dial(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	// Interval well above the post-cancel return budget: a loop that only
	// notices cancel on the next tick (old `for range ticker.C` + Write
	// failing on a dead ctx) misses the budget.
	const interval = 500 * time.Millisecond
	const returnBudget = 100 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	f := &KucoinFetcher{Ws: ws}

	done := make(chan struct{})
	go func() {
		f.pingJob(ctx, interval)
		close(done)
	}()

	deadline := time.Now().Add(5 * interval)
	for pings.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("no ping received before cancel")
		}
		time.Sleep(5 * time.Millisecond)
	}

	cancel()

	select {
	case <-done:
	case <-time.After(returnBudget):
		t.Fatal("pingJob did not return promptly after context cancel")
	}

	afterCancel := pings.Load()
	time.Sleep(interval + 100*time.Millisecond)
	if got := pings.Load(); got != afterCancel {
		t.Fatalf("ping written after cancel: %d -> %d", afterCancel, got)
	}
}
