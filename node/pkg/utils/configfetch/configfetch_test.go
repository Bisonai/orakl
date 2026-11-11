package configfetch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validBundle = `[{"name":"BTC-USDT","feeds":[]}]`

// fastPolicy shrinks the retry policy and clears the env for one test.
func fastPolicy(t *testing.T) {
	t.Helper()
	origRounds, origBackoff, origTimeout := rounds, backoff, timeout
	rounds, backoff, timeout = 3, time.Millisecond, 100*time.Millisecond
	t.Cleanup(func() { rounds, backoff, timeout = origRounds, origBackoff, origTimeout })
	t.Setenv("CONFIG_BASE_URL", "")
	t.Setenv("CONFIG_FALLBACK_URL", "")
	t.Setenv("CONFIG_CACHE_DIR", "")
}

// server answers with the given handlers in order (the last one repeats) and
// counts requests.
func server(t *testing.T, handlers ...http.HandlerFunc) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := int(n.Add(1)) - 1
		if i >= len(handlers) {
			i = len(handlers) - 1
		}
		handlers[i](w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, &n
}

func status(code int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(code) }
}

func body(s string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(s)) }
}

func stall(w http.ResponseWriter, r *http.Request) {
	select {
	case <-r.Context().Done():
	case <-time.After(time.Second):
	}
}

func TestDefaults(t *testing.T) {
	fastPolicy(t)
	os.Unsetenv("CONFIG_BASE_URL")
	os.Unsetenv("CONFIG_FALLBACK_URL")
	assert.Equal(t, []string{DefaultBaseURL, DefaultFallbackURL}, baseURLs())
	assert.Equal(t, "https://config.orakl.network/mainnet_feeds.json", URL(FeedsPath("mainnet")))
	assert.Equal(t, "https://config.orakl.network/kairos_feeds.json", URL(FeedsPath("kairos")))

	t.Setenv("CONFIG_BASE_URL", "http://primary/")
	t.Setenv("CONFIG_FALLBACK_URL", "")
	assert.Equal(t, []string{"http://primary"}, baseURLs())
}

func TestRetriesTransientThenSucceeds(t *testing.T) {
	fastPolicy(t)
	rounds = 4
	srv, n := server(t, status(503), status(429), stall, body(validBundle))
	t.Setenv("CONFIG_BASE_URL", srv.URL)

	got, err := Get[[]map[string]any](context.Background(), FeedsPath("mainnet"))
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, int32(4), n.Load())
}

func TestFallbackUsedWhenPrimaryFails(t *testing.T) {
	fastPolicy(t)
	primary, pn := server(t, status(502))
	fallback, fn := server(t, body(validBundle))
	t.Setenv("CONFIG_BASE_URL", primary.URL)
	t.Setenv("CONFIG_FALLBACK_URL", fallback.URL)

	b, err := Fetch(context.Background(), "/mag7/mainnet/AAPL.json")
	require.NoError(t, err)
	assert.JSONEq(t, validBundle, string(b))
	assert.Equal(t, int32(1), pn.Load())
	assert.Equal(t, int32(1), fn.Load())
}

func TestFallbackUsedOnInvalidJSON(t *testing.T) {
	fastPolicy(t)
	primary, _ := server(t, body("<html>error</html>"))
	fallback, _ := server(t, body(validBundle))
	t.Setenv("CONFIG_BASE_URL", primary.URL)
	t.Setenv("CONFIG_FALLBACK_URL", fallback.URL)

	b, err := Fetch(context.Background(), FeedsPath("mainnet"))
	require.NoError(t, err)
	assert.JSONEq(t, validBundle, string(b))
}

func TestAllTransientFailsAfterRounds(t *testing.T) {
	fastPolicy(t)
	srv, n := server(t, status(502))
	t.Setenv("CONFIG_BASE_URL", srv.URL)

	_, err := Fetch(context.Background(), FeedsPath("mainnet"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "round 3/3")
	assert.Equal(t, int32(3), n.Load())
}

func TestNotFoundIsNotRetried(t *testing.T) {
	fastPolicy(t)
	srv, n := server(t, status(404))
	t.Setenv("CONFIG_BASE_URL", srv.URL)

	_, err := Fetch(context.Background(), FeedsPath("mainnet"))
	require.Error(t, err)
	assert.Equal(t, int32(1), n.Load())
}

func TestInvalidJSONIsNotRetried(t *testing.T) {
	fastPolicy(t)
	srv, n := server(t, body("not json"))
	t.Setenv("CONFIG_BASE_URL", srv.URL)

	_, err := Fetch(context.Background(), FeedsPath("mainnet"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
	assert.Equal(t, int32(1), n.Load())
}

func TestCacheWrittenAndServedOnFailure(t *testing.T) {
	fastPolicy(t)
	dir := t.TempDir()
	t.Setenv("CONFIG_CACHE_DIR", dir)

	ok, _ := server(t, body(validBundle))
	t.Setenv("CONFIG_BASE_URL", ok.URL)
	_, err := Fetch(context.Background(), "/mag7/mainnet/AAPL.json")
	require.NoError(t, err)
	cached, err := os.ReadFile(filepath.Join(dir, "mag7_mainnet_AAPL.json"))
	require.NoError(t, err)
	assert.JSONEq(t, validBundle, string(cached))

	down, _ := server(t, status(503))
	t.Setenv("CONFIG_BASE_URL", down.URL)
	b, err := Fetch(context.Background(), "/mag7/mainnet/AAPL.json")
	require.NoError(t, err)
	assert.JSONEq(t, validBundle, string(b))

	_, err = Fetch(context.Background(), FeedsPath("mainnet"))
	require.Error(t, err, "no cache for a path never fetched")
}

func TestCtxCancelDuringBackoff(t *testing.T) {
	fastPolicy(t)
	backoff = time.Hour
	srv, _ := server(t, status(503))
	t.Setenv("CONFIG_BASE_URL", srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	_, err := Fetch(ctx, FeedsPath("mainnet"))
	assert.True(t, errors.Is(err, context.Canceled), "got %v", err)
}

func TestInvalidInputs(t *testing.T) {
	fastPolicy(t)
	_, err := Fetch(context.Background(), "mainnet_feeds.json")
	assert.Error(t, err)

	for _, u := range []string{"example.com", "/path-only", "ftp://example.com", "mailto:x@example.com"} {
		t.Setenv("CONFIG_BASE_URL", u)
		_, err := Fetch(context.Background(), FeedsPath("mainnet"))
		assert.Error(t, err, u)
		assert.Contains(t, err.Error(), "invalid config base url", u)
	}
}
