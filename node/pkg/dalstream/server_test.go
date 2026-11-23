package dalstream

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func newTestServer(t *testing.T, cfg Config, symbols ...string) (*Hub, *httptest.Server) {
	t.Helper()
	h := newTestHub(t, cfg, symbols...)
	srv := httptest.NewServer(NewServer(h, cfg))
	t.Cleanup(srv.Close)
	return h, srv
}

func status(t *testing.T, method, url string, header http.Header) (*http.Response, string) {
	t.Helper()
	req, _ := http.NewRequest(method, url, nil)
	for k, v := range header {
		req.Header[k] = v
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

func TestStreamSnapshotAndUpdates(t *testing.T) {
	h, srv := newTestServer(t, testConfig(), "BTC-USDT", "ETH-USDT")
	_ = h.Apply(msg("BTC-USDT", "6141018000000", 1000))
	h.Tick()

	r, resp := openStream(t, srv.URL+"/stream?symbols=BTC-USDT", nil)
	if r == nil {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q", ct)
	}
	ev, ok := r.next(t, time.Second)
	if !ok || ev.Event != "snapshot" {
		t.Fatalf("want snapshot, got %+v", ev)
	}
	if p, _ := ev.Data.price(t, "BTC-USDT"); p != "61410.18" {
		t.Fatalf("snapshot price %s", p)
	}

	_ = h.Apply(msg("BTC-USDT", "6141100000000", 1400))
	_ = h.Apply(msg("ETH-USDT", "250000000000", 1400))
	h.Tick()
	ev, ok = r.next(t, time.Second)
	if !ok || ev.Event != "update" || len(ev.Data.D) != 1 {
		t.Fatalf("want BTC update, got %+v", ev)
	}
	if p, at := ev.Data.price(t, "BTC-USDT"); p != "61411" || at != 1400 {
		t.Fatalf("update %s @ %d", p, at)
	}
}

func TestStreamRetryAndPing(t *testing.T) {
	cfg := testConfig()
	cfg.PingInterval = 20 * time.Millisecond
	_, srv := newTestServer(t, cfg, "BTC-USDT")
	r, resp := openStream(t, srv.URL+"/stream", nil)
	if r == nil {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var sawRetry, sawPing bool
	deadline := time.After(2 * time.Second)
	for !sawRetry || !sawPing {
		select {
		case line := <-r.lines:
			sawRetry = sawRetry || line == "retry: 3000"
			sawPing = sawPing || line == ": ping"
		case <-deadline:
			t.Fatalf("retry=%v ping=%v", sawRetry, sawPing)
		}
	}
}

func TestStreamUnknownSymbol(t *testing.T) {
	_, srv := newTestServer(t, testConfig(), "BTC-USDT")
	resp, body := status(t, http.MethodGet, srv.URL+"/stream?symbols=BTC-USDT,NOPE-USDT", nil)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "NOPE-USDT") {
		t.Fatalf("got %d %q", resp.StatusCode, body)
	}
}

func TestStreamNotReady(t *testing.T) {
	cfg := testConfig()
	h := NewHub(cfg)
	srv := httptest.NewServer(NewServer(h, cfg))
	defer srv.Close()
	resp, _ := status(t, http.MethodGet, srv.URL+"/stream", nil)
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("got %d", resp.StatusCode)
	}
}

func TestStreamLimitsAndTrustedProxy(t *testing.T) {
	cfg := testConfig()
	cfg.MaxConns = 4
	cfg.MaxConnsPerIP = 2
	cfg.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}
	h, srv := newTestServer(t, cfg, "BTC-USDT")

	xff := func(ip string) http.Header { return http.Header{"X-Forwarded-For": {ip}} }
	open := func(h http.Header) int {
		r, resp := openStream(t, srv.URL+"/stream", h)
		if r == nil {
			resp.Body.Close()
			return resp.StatusCode
		}
		return http.StatusOK
	}
	// trusted peer: X-Forwarded-For decides; the left (spoofable) hop is ignored
	if got := open(xff("9.9.9.9, 1.1.1.1")); got != 200 {
		t.Fatal(got)
	}
	if got := open(xff("8.8.8.8, 1.1.1.1")); got != 200 {
		t.Fatal(got)
	}
	if got := open(xff("7.7.7.7, 1.1.1.1")); got != http.StatusTooManyRequests {
		t.Fatalf("per-IP limit: %d", got)
	}
	if got := open(xff("2.2.2.2")); got != 200 {
		t.Fatal(got)
	}
	if got := open(xff("3.3.3.3")); got != 200 {
		t.Fatal(got)
	}
	if got := open(xff("4.4.4.4")); got != http.StatusServiceUnavailable {
		t.Fatalf("total limit: %d", got)
	}
	waitFor(t, time.Second, "4 clients", func() bool {
		h.mu.Lock()
		defer h.mu.Unlock()
		return h.total == 4 && h.perIP["1.1.1.1"] == 2
	})
}

func TestStreamUntrustedPeerIgnoresXFF(t *testing.T) {
	cfg := testConfig()
	cfg.MaxConnsPerIP = 1
	_, srv := newTestServer(t, cfg, "BTC-USDT")
	r, resp := openStream(t, srv.URL+"/stream", http.Header{"X-Forwarded-For": {"1.1.1.1"}})
	if r == nil {
		t.Fatal(resp.StatusCode)
	}
	// a different X-Forwarded-For does not help: the peer (127.0.0.1) is not trusted
	_, resp = openStream(t, srv.URL+"/stream", http.Header{"X-Forwarded-For": {"2.2.2.2"}})
	resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("got %d", resp.StatusCode)
	}
}

func TestClientIP(t *testing.T) {
	proxies, err := ParseTrustedProxies("10.0.0.0/8, 192.168.1.1")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		remote string
		xff    []string
		want   string
	}{
		{"1.2.3.4:5", []string{"9.9.9.9"}, "1.2.3.4"},           // untrusted peer
		{"10.0.0.1:5", nil, "10.0.0.1"},                         // trusted, no header
		{"10.0.0.1:5", []string{"9.9.9.9, 5.5.5.5"}, "5.5.5.5"}, // rightmost untrusted
		{"10.0.0.1:5", []string{"5.5.5.5, 192.168.1.1"}, "5.5.5.5"},
		{"10.0.0.1:5", []string{"9.9.9.9", "5.5.5.5"}, "5.5.5.5"}, // multiple headers
		{"10.0.0.1:5", []string{"10.0.0.2"}, "10.0.0.2"},          // all trusted
		{"10.0.0.1:5", []string{"5.5.5.5, junk"}, "10.0.0.1"},
		{"[::ffff:1.2.3.4]:5", nil, "1.2.3.4"},
	}
	for _, c := range cases {
		if got := clientIP(c.remote, c.xff, proxies); got != c.want {
			t.Errorf("clientIP(%q, %q) = %q, want %q", c.remote, c.xff, got, c.want)
		}
	}
	if _, err := ParseTrustedProxies("not-an-ip"); err == nil {
		t.Fatal("bad proxy accepted")
	}

	mapped, err := ParseTrustedProxies("::ffff:10.0.0.0/104")
	if err != nil {
		t.Fatal(err)
	}
	if got := clientIP("10.0.0.5:1", []string{"5.5.5.5"}, mapped); got != "5.5.5.5" {
		t.Fatalf("IPv4-mapped proxy prefix: got %q", got)
	}

	for in, want := range map[string]string{
		"1.2.3.4":            "1.2.3.4",
		"2001:db8:1:2:3::9":  "2001:db8:1:2::/64",
		"2001:db8:1:2:ff::1": "2001:db8:1:2::/64",
	} {
		if got := limitKey(in); got != want {
			t.Errorf("limitKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCORSAllowlist(t *testing.T) {
	cfg := testConfig()
	cfg.AllowedOrigins = ParseOrigins("https://orakl.network, https://www.orakl.network/")
	_, srv := newTestServer(t, cfg, "BTC-USDT")

	r, resp := openStream(t, srv.URL+"/stream", http.Header{"Origin": {"https://www.orakl.network"}})
	if r == nil {
		t.Fatal(resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "https://www.orakl.network" {
		t.Fatalf("ACAO = %q", got)
	}
	if got := resp.Header.Get("Vary"); got != "Origin" {
		t.Fatalf("Vary = %q", got)
	}

	resp, _ = status(t, http.MethodGet, srv.URL+"/stream", http.Header{"Origin": {"https://evil.example"}})
	if resp.StatusCode != http.StatusForbidden || resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("other origin: %d %q", resp.StatusCode, resp.Header.Get("Access-Control-Allow-Origin"))
	}

	resp, _ = status(t, http.MethodOptions, srv.URL+"/stream", http.Header{
		"Origin":                        {"https://orakl.network"},
		"Access-Control-Request-Method": {"GET"},
	})
	if resp.StatusCode != http.StatusNoContent || resp.Header.Get("Access-Control-Allow-Origin") != "https://orakl.network" ||
		resp.Header.Get("Access-Control-Allow-Methods") != "GET, OPTIONS" {
		t.Fatalf("preflight: %d %v", resp.StatusCode, resp.Header)
	}

	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodHead} {
		resp, _ = status(t, m, srv.URL+"/stream", nil)
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("%s: %d", m, resp.StatusCode)
		}
	}

	// no Origin (curl, server-side): allowed, no CORS header
	resp, _ = status(t, http.MethodGet, srv.URL+"/health", nil)
	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("ACAO without Origin")
	}
}

func TestHealthEndpoint(t *testing.T) {
	cfg := testConfig()
	cfg.StaleAfter = time.Second
	h, srv := newTestServer(t, cfg, "BTC-USDT")
	now := time.Unix(1000, 0)
	h.mu.Lock()
	h.now = func() time.Time { return now }
	h.mu.Unlock()

	if resp, _ := status(t, http.MethodGet, srv.URL+"/health", nil); resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("disconnected: %d", resp.StatusCode)
	}
	h.SetConnected(true)
	_ = h.Apply(msg("BTC-USDT", "1", 1))
	if resp, body := status(t, http.MethodGet, srv.URL+"/health", nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("healthy: %d %s", resp.StatusCode, body)
	}
	h.mu.Lock()
	now = now.Add(2 * time.Second)
	h.mu.Unlock()
	if resp, body := status(t, http.MethodGet, srv.URL+"/health", nil); resp.StatusCode != http.StatusServiceUnavailable || !strings.Contains(body, "stale") {
		t.Fatalf("stale: %d %s", resp.StatusCode, body)
	}
}
