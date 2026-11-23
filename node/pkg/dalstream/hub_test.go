package dalstream

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFormatPrice(t *testing.T) {
	cases := []struct {
		raw  string
		dec  int
		want string
	}{
		{"6141018000000", 8, "61410.18"},
		{"6141018123456", 8, "61410.18123456"},
		{"100000000", 8, "1"},
		{"1", 8, "0.00000001"},
		{"0", 8, "0"},
		{"000123", 2, "1.23"},
		{"12345", 0, "12345"},
		{"-150", 2, "-1.5"},
		{"-0", 2, "0"},
	}
	for _, c := range cases {
		got, err := formatPrice(c.raw, c.dec)
		if err != nil || got != c.want {
			t.Errorf("formatPrice(%q, %d) = %q, %v; want %q", c.raw, c.dec, got, err, c.want)
		}
	}
	for _, bad := range []string{"", "-", "1.5", "1e5", "abc"} {
		if _, err := formatPrice(bad, 8); err == nil {
			t.Errorf("formatPrice(%q) want error", bad)
		}
	}
	if _, err := formatPrice("1", -1); err == nil {
		t.Error("negative decimals want error")
	}
}

func newTestHub(t *testing.T, cfg Config, symbols ...string) *Hub {
	t.Helper()
	h := NewHub(cfg)
	h.SetSymbols(symbols)
	return h
}

func register(t *testing.T, h *Hub, ip, query string) (*client, context.Context) {
	t.Helper()
	key, symbols, err := h.ParseFilter(query)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	c, err := h.Register(ip, key, symbols, cancel)
	if err != nil {
		t.Fatal(err)
	}
	return c, ctx
}

func drain(t *testing.T, c *client) []sseEvent {
	t.Helper()
	var out []sseEvent
	for {
		select {
		case b := <-c.queue:
			out = append(out, parseEvent(t, b))
		default:
			return out
		}
	}
}

func TestTickSendsOnlyChangedValues(t *testing.T) {
	h := newTestHub(t, testConfig(), "BTC-USDT", "ETH-USDT", "KAIA-USDT")
	apply := func(s, v string, at int64) {
		if err := h.Apply(msg(s, v, at)); err != nil {
			t.Fatal(err)
		}
	}
	apply("BTC-USDT", "6141018000000", 1000)
	apply("ETH-USDT", "250000000000", 1000)

	c, _ := register(t, h, "1.1.1.1", "")
	if evs := drain(t, c); len(evs) != 1 || evs[0].Event != "snapshot" {
		t.Fatalf("want one snapshot, got %+v", evs)
	}

	// first tick publishes the values seen so far
	h.Tick()
	evs := drain(t, c)
	if len(evs) != 1 || evs[0].Event != "update" || len(evs[0].Data.D) != 2 {
		t.Fatalf("first tick: %+v", evs)
	}

	// new aggregateTime, same value: nothing to send
	apply("BTC-USDT", "6141018000000", 1400)
	apply("ETH-USDT", "250000000000", 1400)
	h.Tick()
	if evs := drain(t, c); len(evs) != 0 {
		t.Fatalf("unchanged values sent: %+v", evs)
	}

	// nothing at all
	h.Tick()
	if evs := drain(t, c); len(evs) != 0 {
		t.Fatalf("empty tick sent: %+v", evs)
	}

	// BTC changes twice, ETH goes away and back to the same value, KAIA is new
	apply("BTC-USDT", "6141100000000", 1800)
	apply("BTC-USDT", "6141200000000", 2200)
	apply("ETH-USDT", "250100000000", 1800)
	apply("ETH-USDT", "250000000000", 2200)
	apply("KAIA-USDT", "12345678", 2200)
	h.Tick()
	evs = drain(t, c)
	if len(evs) != 1 {
		t.Fatalf("want one update, got %+v", evs)
	}
	d := evs[0].Data
	if len(d.D) != 2 {
		t.Fatalf("want BTC and KAIA only, got %v", d.D)
	}
	if p, at := d.price(t, "BTC-USDT"); p != "61412" || at != 2200 {
		t.Fatalf("BTC = %s @ %d", p, at)
	}
	if p, _ := d.price(t, "KAIA-USDT"); p != "0.12345678" {
		t.Fatalf("KAIA = %s", p)
	}
	if ts, _ := d.T.Int64(); ts <= 0 {
		t.Fatalf("bad t %v", d.T)
	}
}

func TestSnapshotOnRegister(t *testing.T) {
	h := newTestHub(t, testConfig(), "BTC-USDT", "ETH-USDT", "KAIA-USDT")
	_ = h.Apply(msg("BTC-USDT", "6141018000000", 1000))
	_ = h.Apply(msg("ETH-USDT", "250000000000", 1100))
	h.Tick()
	// not yet published: the next update carries it
	_ = h.Apply(msg("BTC-USDT", "6141100000000", 1400))

	c, _ := register(t, h, "1.1.1.1", "BTC-USDT,KAIA-USDT")
	evs := drain(t, c)
	if len(evs) != 1 || evs[0].Event != "snapshot" {
		t.Fatalf("want snapshot, got %+v", evs)
	}
	// KAIA has no data yet; ETH is not requested
	if len(evs[0].Data.D) != 1 {
		t.Fatalf("snapshot = %v", evs[0].Data.D)
	}
	if p, at := evs[0].Data.price(t, "BTC-USDT"); p != "61410.18" || at != 1000 {
		t.Fatalf("BTC = %s @ %d", p, at)
	}
	want := `event: snapshot` + "\n" + `data: {"t":`
	if evs[0].Raw[:len(want)] != want {
		t.Fatalf("raw = %q", evs[0].Raw)
	}
	h.Tick()
	evs = drain(t, c)
	if len(evs) != 1 || evs[0].Event != "update" {
		t.Fatalf("want update, got %+v", evs)
	}
	if p, _ := evs[0].Data.price(t, "BTC-USDT"); p != "61411" {
		t.Fatalf("BTC = %s", p)
	}
}

// A client that connects between ticks must not keep a value that reverted
// before the next tick.
func TestSnapshotConsistentWithTicks(t *testing.T) {
	h := newTestHub(t, testConfig(), "BTC-USDT")
	_ = h.Apply(msg("BTC-USDT", "100", 1))
	h.Tick()
	_ = h.Apply(msg("BTC-USDT", "200", 2))
	c, _ := register(t, h, "1.1.1.1", "")
	evs := drain(t, c)
	if p, _ := evs[0].Data.price(t, "BTC-USDT"); p != "0.000001" {
		t.Fatalf("snapshot = %s, want the published value", p)
	}
	_ = h.Apply(msg("BTC-USDT", "100", 3))
	h.Tick()
	if evs := drain(t, c); len(evs) != 0 {
		t.Fatalf("reverted value sent: %+v", evs)
	}
}

func TestSymbolFilter(t *testing.T) {
	h := newTestHub(t, testConfig(), "BTC-USDT", "ETH-USDT")
	key, symbols, err := h.ParseFilter(" ETH-USDT,BTC-USDT,ETH-USDT ")
	if err != nil || key != "BTC-USDT,ETH-USDT" || len(symbols) != 2 {
		t.Fatalf("got %q %v %v", key, symbols, err)
	}
	if key, symbols, err := h.ParseFilter(""); err != nil || key != "" || symbols != nil {
		t.Fatalf("empty = %q %v %v", key, symbols, err)
	}
	if _, _, err := h.ParseFilter("BTC-USDT,NOPE-USDT"); err == nil {
		t.Fatal("unknown symbol accepted")
	}

	only, _ := register(t, h, "1.1.1.1", "ETH-USDT")
	drain(t, only)
	_ = h.Apply(msg("BTC-USDT", "1", 1))
	h.Tick()
	if evs := drain(t, only); len(evs) != 0 {
		t.Fatalf("unrequested symbol sent: %+v", evs)
	}
	_ = h.Apply(msg("ETH-USDT", "2", 2))
	h.Tick()
	if evs := drain(t, only); len(evs) != 1 || len(evs[0].Data.D) != 1 {
		t.Fatalf("got %+v", evs)
	}
}

func TestSameFilterSharesPayload(t *testing.T) {
	h := newTestHub(t, testConfig(), "BTC-USDT", "ETH-USDT")
	a, _ := register(t, h, "1.1.1.1", "BTC-USDT,ETH-USDT")
	b, _ := register(t, h, "2.2.2.2", "ETH-USDT,BTC-USDT")
	if a.group != b.group {
		t.Fatal("same filter in different order should share a group")
	}
	drain(t, a)
	drain(t, b)
	_ = h.Apply(msg("BTC-USDT", "1", 1))
	h.Tick()
	pa, pb := <-a.queue, <-b.queue
	if &pa[0] != &pb[0] {
		t.Fatal("payload encoded per client")
	}
}

func TestConnectionLimits(t *testing.T) {
	cfg := testConfig()
	cfg.MaxConns = 3
	cfg.MaxConnsPerIP = 2
	h := newTestHub(t, cfg, "BTC-USDT")
	reg := func(ip string) (*client, error) {
		return h.Register(ip, "", nil, func() {})
	}
	a1, _ := reg("1.1.1.1")
	if _, err := reg("1.1.1.1"); err != nil {
		t.Fatal(err)
	}
	if _, err := reg("1.1.1.1"); !errors.Is(err, errMaxPerIP) {
		t.Fatalf("per-IP limit: %v", err)
	}
	if _, err := reg("2.2.2.2"); err != nil {
		t.Fatal(err)
	}
	if _, err := reg("3.3.3.3"); !errors.Is(err, errMaxConnections) {
		t.Fatalf("total limit: %v", err)
	}
	h.Unregister(a1)
	h.Unregister(a1) // idempotent
	if _, err := reg("3.3.3.3"); err != nil {
		t.Fatalf("slot not freed: %v", err)
	}
	if _, err := reg("1.1.1.1"); !errors.Is(err, errMaxConnections) {
		t.Fatalf("total limit: %v", err)
	}
}

func TestSlowClientDropped(t *testing.T) {
	cfg := testConfig()
	cfg.ClientQueue = 2
	h := newTestHub(t, cfg, "BTC-USDT")
	slow, slowCtx := register(t, h, "1.1.1.1", "") // never reads; snapshot fills 1 of 2
	fast, _ := register(t, h, "2.2.2.2", "")
	for i := int64(1); i <= 3; i++ {
		_ = h.Apply(msg("BTC-USDT", jsonInt(i), i))
		h.Tick()
		drain(t, fast)
	}
	select {
	case <-slowCtx.Done():
	default:
		t.Fatal("slow client not dropped")
	}
	if !slow.removed || fast.removed {
		t.Fatalf("removed: slow=%v fast=%v", slow.removed, fast.removed)
	}
	if h.total != 1 || h.perIP["1.1.1.1"] != 0 {
		t.Fatalf("counts not released: total=%d perIP=%v", h.total, h.perIP)
	}
}

func TestHealthStale(t *testing.T) {
	h := newTestHub(t, testConfig(), "BTC-USDT")
	now := time.Unix(1000, 0)
	h.now = func() time.Time { return now }
	if ok, _ := h.Healthy(time.Second); ok {
		t.Fatal("healthy while disconnected")
	}
	h.SetConnected(true)
	if ok, _ := h.Healthy(time.Second); ok {
		t.Fatal("healthy without data")
	}
	_ = h.Apply(msg("BTC-USDT", "1", 1))
	if ok, reason := h.Healthy(time.Second); !ok {
		t.Fatalf("unhealthy: %s", reason)
	}
	// a new connection is unhealthy until it delivers data
	h.SetConnected(false)
	h.SetConnected(true)
	if ok, reason := h.Healthy(time.Second); ok || reason != "no upstream data yet" {
		t.Fatalf("reconnected without data: ok=%v %s", ok, reason)
	}
	_ = h.Apply(msg("BTC-USDT", "1", 2))
	now = now.Add(2 * time.Second)
	if ok, reason := h.Healthy(time.Second); ok || reason != "upstream stale" {
		t.Fatalf("stalled upstream: ok=%v %s", ok, reason)
	}
	if age := h.lastMessageAge(); age != 2 {
		t.Fatalf("age = %v", age)
	}
}

func TestSetSymbolsRemovesData(t *testing.T) {
	h := newTestHub(t, testConfig(), "BTC-USDT", "ETH-USDT")
	_ = h.Apply(msg("ETH-USDT", "1", 1))
	if added := h.SetSymbols([]string{"BTC-USDT", "KAIA-USDT"}); len(added) != 1 || added[0] != "KAIA-USDT" {
		t.Fatalf("added = %v", added)
	}
	if _, ok := h.latest["ETH-USDT"]; ok {
		t.Fatal("removed symbol kept")
	}
	if _, _, err := h.ParseFilter("ETH-USDT"); err == nil {
		t.Fatal("removed symbol accepted")
	}
	_ = h.Apply(msg("ETH-USDT", "2", 2))
	if _, ok := h.latest["ETH-USDT"]; ok {
		t.Fatal("data for unknown symbol stored")
	}
	if added := h.SetSymbols(nil); added != nil || len(h.known) != 2 {
		t.Fatal("empty list must be ignored")
	}
}
