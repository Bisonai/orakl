package dalstream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nhooyr.io/websocket"
)

func startUpstream(t *testing.T, cfg Config, dal *fakeDAL) (*Hub, *httptest.Server) {
	t.Helper()
	dalSrv := httptest.NewServer(dal)
	t.Cleanup(dalSrv.Close)
	cfg.DALURL = dalSrv.URL
	h := NewHub(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); NewUpstream(cfg, h).Run(ctx) }()
	t.Cleanup(func() { cancel(); <-done })
	return h, dalSrv
}

func nextConn(t *testing.T, dal *fakeDAL) *websocket.Conn {
	t.Helper()
	select {
	case c := <-dal.connected:
		return c
	case <-time.After(3 * time.Second):
		t.Fatal("upstream did not connect")
		return nil
	}
}

func TestUpstreamSubscribesAndApplies(t *testing.T) {
	dal := newFakeDAL("BTC-USDT", "ETH-USDT")
	h, _ := startUpstream(t, testConfig(), dal)
	c := nextConn(t, dal)
	waitFor(t, time.Second, "subscription", func() bool {
		return dal.isSubscribed("BTC-USDT") && dal.isSubscribed("ETH-USDT")
	})
	send(t, c, msg("BTC-USDT", "6141018000000", 1000))
	waitFor(t, time.Second, "data applied", func() bool {
		ok, _ := h.Healthy(time.Second)
		return ok
	})
	h.mu.Lock()
	e := h.latest["BTC-USDT"]
	h.mu.Unlock()
	if e == nil || e.price != "61410.18" || e.aggTime != 1000 {
		t.Fatalf("latest = %+v", e)
	}
}

func TestUpstreamReconnectsAfterLoss(t *testing.T) {
	dal := newFakeDAL("BTC-USDT")
	h, _ := startUpstream(t, testConfig(), dal)
	c := nextConn(t, dal)
	waitFor(t, time.Second, "subscription", func() bool { return dal.isSubscribed("BTC-USDT") })
	send(t, c, msg("BTC-USDT", "1", 1))
	waitFor(t, time.Second, "healthy", func() bool { ok, _ := h.Healthy(time.Second); return ok })

	c.CloseNow()
	waitFor(t, time.Second, "disconnect seen", func() bool { ok, _ := h.Healthy(time.Second); return !ok })

	c2 := nextConn(t, dal)
	waitFor(t, time.Second, "connected again", func() bool {
		h.mu.Lock()
		defer h.mu.Unlock()
		return h.connected
	})
	send(t, c2, msg("BTC-USDT", "2", 2))
	waitFor(t, time.Second, "healthy again", func() bool { ok, _ := h.Healthy(time.Second); return ok })
}

func TestUpstreamStallUnhealthyAndReconnects(t *testing.T) {
	cfg := testConfig()
	cfg.StaleAfter = 200 * time.Millisecond
	dal := newFakeDAL("BTC-USDT")
	h, _ := startUpstream(t, cfg, dal)
	c := nextConn(t, dal)
	waitFor(t, time.Second, "subscription", func() bool { return dal.isSubscribed("BTC-USDT") })
	send(t, c, msg("BTC-USDT", "1", 1))
	waitFor(t, time.Second, "healthy", func() bool { ok, _ := h.Healthy(cfg.StaleAfter); return ok })

	// DAL keeps the socket open but stops sending
	waitFor(t, 2*time.Second, "unhealthy on stall", func() bool { ok, _ := h.Healthy(cfg.StaleAfter); return !ok })
	c2 := nextConn(t, dal) // stalled connection dropped and rebuilt
	send(t, c2, msg("BTC-USDT", "2", 2))
	waitFor(t, time.Second, "healthy again", func() bool { ok, _ := h.Healthy(cfg.StaleAfter); return ok })
}

func TestUpstreamRefreshSubscribesNewSymbols(t *testing.T) {
	cfg := testConfig()
	cfg.SymbolsRefresh = 30 * time.Millisecond
	dal := newFakeDAL("BTC-USDT")
	h, _ := startUpstream(t, cfg, dal)
	c := nextConn(t, dal)
	waitFor(t, time.Second, "subscription", func() bool { return dal.isSubscribed("BTC-USDT") })

	dal.setSymbols("BTC-USDT", "KAIA-USDT")
	waitFor(t, 2*time.Second, "new symbol subscribed", func() bool { return dal.isSubscribed("KAIA-USDT") })
	if _, _, err := h.ParseFilter("KAIA-USDT"); err != nil {
		t.Fatal(err)
	}
	send(t, c, msg("KAIA-USDT", "12345678", 1))
	waitFor(t, time.Second, "new symbol applied", func() bool {
		h.mu.Lock()
		defer h.mu.Unlock()
		return h.latest["KAIA-USDT"] != nil
	})
}

func TestUpstreamBadKey(t *testing.T) {
	cfg := testConfig()
	cfg.APIKey = "wrong"
	dal := newFakeDAL("BTC-USDT")
	h, srv := startUpstream(t, cfg, dal)
	time.Sleep(100 * time.Millisecond)
	if ok, _ := h.Healthy(time.Second); ok {
		t.Fatal("healthy with a bad key")
	}
	resp, err := http.Get(srv.URL + "/symbols")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("fake DAL accepted no key: %d", resp.StatusCode)
	}
}

// TestEndToEnd runs the full pipeline: fake DAL -> upstream -> hub ticks ->
// SSE client.
func TestEndToEnd(t *testing.T) {
	cfg := testConfig()
	cfg.Tick = 100 * time.Millisecond
	dal := newFakeDAL("BTC-USDT", "ETH-USDT")
	h, _ := startUpstream(t, cfg, dal)
	c := nextConn(t, dal)
	waitFor(t, time.Second, "subscription", func() bool { return dal.isSubscribed("ETH-USDT") })
	send(t, c, msg("BTC-USDT", "6141018000000", 1000))
	send(t, c, msg("ETH-USDT", "250000000000", 1000))
	waitFor(t, time.Second, "data", func() bool {
		h.mu.Lock()
		defer h.mu.Unlock()
		return len(h.latest) == 2
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.Tick() // publish, so the snapshot has both
	go h.Run(ctx, cfg.Tick)
	srv := httptest.NewServer(NewServer(h, cfg))
	t.Cleanup(srv.Close) // after the stream body is closed (cleanups run LIFO)

	r, resp := openStream(t, srv.URL+"/stream", nil)
	if r == nil {
		t.Fatal(resp.StatusCode)
	}
	ev, _ := r.next(t, time.Second)
	if ev.Event != "snapshot" || len(ev.Data.D) != 2 {
		t.Fatalf("snapshot: %+v", ev)
	}
	// the first tick may republish the initial values; then only ETH moves
	time.Sleep(3 * cfg.Tick)
	for len(r.events) > 0 {
		<-r.events
	}
	send(t, c, msg("BTC-USDT", "6141018000000", 1400))
	send(t, c, msg("ETH-USDT", "251000000000", 1400))
	ev, ok := r.next(t, time.Second)
	if !ok || ev.Event != "update" || len(ev.Data.D) != 1 {
		t.Fatalf("update: %+v", ev)
	}
	if p, _ := ev.Data.price(t, "ETH-USDT"); p != "2510" {
		t.Fatalf("ETH = %s", p)
	}
	if ev, ok := r.next(t, 3*cfg.Tick); ok {
		t.Fatalf("unexpected event with no change: %+v", ev)
	}
}
