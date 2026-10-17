package xt

import (
	"context"
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
			if string(data) == "ping" {
				count.Add(1)
			}
		}
	}))
}

func TestPingStopsOnContextCancel(t *testing.T) {
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

	ctx, cancel := context.WithCancel(context.Background())
	f := &XtFetcher{Ws: ws}

	done := make(chan struct{})
	go func() {
		f.ping(ctx, 10*time.Millisecond)
		close(done)
	}()

	deadline := time.Now().Add(time.Second)
	for pings.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("no ping received before cancel")
		}
		time.Sleep(5 * time.Millisecond)
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("ping goroutine did not return after context cancel")
	}

	afterCancel := pings.Load()
	time.Sleep(50 * time.Millisecond)
	if got := pings.Load(); got != afterCancel {
		t.Fatalf("ping written after cancel: %d -> %d", afterCancel, got)
	}
}
