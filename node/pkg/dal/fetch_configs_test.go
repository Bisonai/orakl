package dal

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	errorsentinel "bisonai.com/miko/node/pkg/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Retry/fallback behaviour is covered in pkg/utils/configfetch; these
// tests only check the DAL wrapper (path + decoding + error propagation).

const validConfigsBody = `[{"id":1,"name":"BTC-USDT","fetchInterval":2000,"aggregateInterval":5000,"submitInterval":15000}]`

// withFetchServer points configfetch at an httptest server (no fallback).
func withFetchServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("CONFIG_BASE_URL", srv.URL)
	t.Setenv("CONFIG_FALLBACK_URL", "")
}

func TestFetchConfigsDecodesBundle(t *testing.T) {
	withFetchServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/mainnet_feeds.json", r.URL.Path)
		_, _ = w.Write([]byte(validConfigsBody))
	})

	configs, err := fetchConfigs(context.Background(), "mainnet")
	require.NoError(t, err)
	require.Len(t, configs, 1)
	assert.Equal(t, "BTC-USDT", configs[0].Name)
}

func TestFetchConfigsNoRetryOn404(t *testing.T) {
	var calls atomic.Int32
	withFetchServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := fetchConfigs(context.Background(), "mainnet")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errorsentinel.ErrRequestStatusNotOk))
	assert.Contains(t, err.Error(), "status 404")
	assert.Equal(t, int32(1), calls.Load())
}
