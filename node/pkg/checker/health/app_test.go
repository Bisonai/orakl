//nolint:all
package health

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/testutil/testenv"
	"github.com/stretchr/testify/assert"
)

// redisURL returns a redis:// URL for REDIS_HOST/REDIS_PORT, falling back to
// the in-process miniredis server when REDIS_HOST is unset.
func redisURL(t *testing.T) string {
	t.Helper()
	testenv.Redis(t)
	return "redis://" + net.JoinHostPort(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT"))
}

func TestSelectHealthCheckJSON(t *testing.T) {
	// mainnet picks the cypress bundle; kairos picks the baobab bundle
	// (the embedded JSON filenames stay literal).
	raw, ok := selectHealthCheckJSON("mainnet")
	assert.True(t, ok, "mainnet should select a bundle")
	assert.True(t, bytes.Equal(raw, cypressJSON), "mainnet should pick cypress_healthcheck.json")

	raw, ok = selectHealthCheckJSON("kairos")
	assert.True(t, ok, "kairos should select a bundle")
	assert.True(t, bytes.Equal(raw, baobabJSON), "kairos should pick baobab_healthcheck.json")

	_, ok = selectHealthCheckJSON("invalid")
	assert.False(t, ok, "unknown chain should not select a bundle")
}

func TestCheckUrl(t *testing.T) {
	ctx := context.Background()
	// Test case 1: URL with "http" prefix
	httpUrl := HealthCheckUrl{Url: "http://example.com"}
	if !checkUrl(ctx, httpUrl) {
		t.Errorf("checkUrl(%s) = false, expected true", httpUrl.Url)
	}

	// Test case 2: URL with "redis" prefix
	redisUrl := HealthCheckUrl{Url: redisURL(t)}
	if !checkUrl(ctx, redisUrl) {
		t.Errorf("checkUrl(%s) = false, expected true", redisUrl.Url)
	}

	// Test case 3: Invalid URL
	invalidUrl := HealthCheckUrl{Url: "invalid-url"}
	if checkUrl(ctx, invalidUrl) {
		t.Errorf("checkUrl(%s) = true, expected false", invalidUrl.Url)
	}
}

func TestCheckHttp(t *testing.T) {
	// Test case 1: Valid URL with HTTP 200 response
	validUrl := "http://example.com"
	if !checkHttp(context.Background(), validUrl) {
		t.Errorf("checkHttp(%s) = false, expected true", validUrl)
	}

	// Test case 2: Valid URL with non-200 response
	non200Url := "http://example.com/nonexistent"
	if checkHttp(context.Background(), non200Url) {
		t.Errorf("checkHttp(%s) = true, expected false", non200Url)
	}

	// Test case 3: Invalid URL
	invalidUrl := "invalid-url"
	if checkHttp(context.Background(), invalidUrl) {
		t.Errorf("checkHttp(%s) = true, expected false", invalidUrl)
	}
}

func TestCheckRedis(t *testing.T) {
	ctx := context.Background()
	url := redisURL(t)

	result := checkRedis(ctx, url)

	assert.True(t, result)
}

func TestRedactURL(t *testing.T) {
	cases := map[string]string{
		"postgresql://user:secret@10.0.0.1:5432/node": "postgresql://user:xxxxx@10.0.0.1:5432/node",
		"http://orakl-api.orakl.svc/health":           "http://orakl-api.orakl.svc/health",
		"postgresql://user:p%zz@host/db":              "<unparsable url>",
	}
	for in, want := range cases {
		if got := redactURL(in); got != want {
			t.Errorf("redactURL(%q) = %q, want %q", in, got, want)
		}
	}
}

// silentListener accepts TCP connections and never answers.
func silentListener(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		var conns []net.Conn
		defer func() {
			for _, c := range conns {
				c.Close()
			}
		}()
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			conns = append(conns, c)
		}
	}()
	return ln.Addr().String()
}

func withCheckTimeout(t *testing.T, d time.Duration) {
	t.Helper()
	prev := checkTimeout
	checkTimeout = d
	t.Cleanup(func() { checkTimeout = prev })
}

func assertTimesOut(t *testing.T, target HealthCheckUrl) {
	t.Helper()
	done := make(chan bool, 1)
	go func() { done <- checkUrl(context.Background(), target) }()
	select {
	case alive := <-done:
		assert.False(t, alive, "hung target must be reported down")
	case <-time.After(5 * time.Second):
		t.Fatalf("check of %s did not time out", target.Name)
	}
}

func TestCheckUrlHungHttpTimesOut(t *testing.T) {
	withCheckTimeout(t, 200*time.Millisecond)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })
	assertTimesOut(t, HealthCheckUrl{Name: "http", Url: srv.URL})
}

func TestCheckUrlHungPostgresTimesOut(t *testing.T) {
	withCheckTimeout(t, 200*time.Millisecond)
	assertTimesOut(t, HealthCheckUrl{Name: "pg", Url: "postgresql://u:p@" + silentListener(t) + "/db?sslmode=disable&connect_timeout=0"})
}

func TestCheckUrlHungRedisTimesOut(t *testing.T) {
	withCheckTimeout(t, 200*time.Millisecond)
	assertTimesOut(t, HealthCheckUrl{Name: "redis", Url: "redis://" + silentListener(t)})
}
