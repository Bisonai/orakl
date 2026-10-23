package dal

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	errorsentinel "bisonai.com/miko/node/pkg/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validConfigsBody = `[{"id":1,"name":"BTC-USDT","fetchInterval":2000,"aggregateInterval":5000,"submitInterval":15000}]`

// withFetchServer points fetchConfigs at an httptest server and shrinks
// the retry policy so tests don't sleep.
func withFetchServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	origURL, origAttempts, origBackoff, origTimeout := baseMikoConfigUrl, fetchConfigsAttempts, fetchConfigsBackoff, fetchConfigsTimeout
	baseMikoConfigUrl = srv.URL + "/%s_feeds.json"
	fetchConfigsAttempts = 5
	fetchConfigsBackoff = time.Millisecond
	fetchConfigsTimeout = 100 * time.Millisecond
	t.Cleanup(func() {
		baseMikoConfigUrl, fetchConfigsAttempts, fetchConfigsBackoff, fetchConfigsTimeout = origURL, origAttempts, origBackoff, origTimeout
	})
}

// Retries 503, 429 and a stalled response, then succeeds on attempt 4.
func TestFetchConfigsRetriesThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	withFetchServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/mainnet_feeds.json", r.URL.Path)
		switch calls.Add(1) {
		case 1:
			w.WriteHeader(http.StatusServiceUnavailable)
		case 2:
			w.WriteHeader(http.StatusTooManyRequests)
		case 3:
			// Stall past the per-attempt client timeout.
			select {
			case <-r.Context().Done():
			case <-time.After(2 * time.Second):
			}
		default:
			_, _ = w.Write([]byte(validConfigsBody))
		}
	})

	configs, err := fetchConfigs(context.Background(), "mainnet")
	require.NoError(t, err)
	require.Len(t, configs, 1)
	assert.Equal(t, "BTC-USDT", configs[0].Name)
	assert.Equal(t, int32(4), calls.Load())
}

func TestFetchConfigsAllAttemptsFail(t *testing.T) {
	var calls atomic.Int32
	withFetchServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	})

	_, err := fetchConfigs(context.Background(), "mainnet")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errorsentinel.ErrRequestStatusNotOk))
	assert.Contains(t, err.Error(), "attempt 5/5")
	assert.Equal(t, int32(5), calls.Load())
}

func TestFetchConfigsNoRetryOn404(t *testing.T) {
	var calls atomic.Int32
	withFetchServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := fetchConfigs(context.Background(), "mainnet")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 404")
	assert.Equal(t, int32(1), calls.Load())
}

func TestFetchConfigsNoRetryOnBadJSON(t *testing.T) {
	var calls atomic.Int32
	withFetchServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte("not json"))
	})

	_, err := fetchConfigs(context.Background(), "mainnet")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not json")
	assert.Equal(t, int32(1), calls.Load())
}

func TestFetchConfigsStopsOnContextCancel(t *testing.T) {
	var calls atomic.Int32
	withFetchServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	fetchConfigsBackoff = time.Hour

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel well after the first (fast, local) attempt returned, i.e.
	// while fetchConfigs waits out the 1h backoff.
	time.AfterFunc(50*time.Millisecond, cancel)

	_, err := fetchConfigs(ctx, "mainnet")
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, int32(1), calls.Load())
}

func TestFetchConfigsInvalidURLNoRetry(t *testing.T) {
	var calls atomic.Int32
	withFetchServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
	})

	for _, u := range []string{"not-a-url/%s_feeds.json", "/%s_feeds.json", "localhost:8080/%s_feeds.json", "ftp://host/%s_feeds.json"} {
		baseMikoConfigUrl = u
		_, err := fetchConfigs(context.Background(), "mainnet")
		require.Error(t, err, u)
		assert.Contains(t, err.Error(), "invalid config url", u)
	}
	assert.Equal(t, int32(0), calls.Load())
}
