package xt

import (
	"context"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/wss"
)

func TestPingStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	f := &XtFetcher{Ws: &wss.WebsocketHelper{}}

	done := make(chan struct{})
	go func() {
		f.ping(ctx, 10*time.Millisecond)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("ping goroutine did not return after context cancel")
	}
}
