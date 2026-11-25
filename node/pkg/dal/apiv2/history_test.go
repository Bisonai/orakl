package apiv2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/common/types"
	"bisonai.com/orakl/node/pkg/dal/collector"
	"bisonai.com/orakl/node/pkg/dal/history"
	"bisonai.com/orakl/node/pkg/dal/hub"
	"bisonai.com/orakl/node/pkg/dal/utils/keycache"
	"bisonai.com/orakl/node/pkg/dal/utils/keystore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const otherSymbol = "ETH-USDT"

func newHistoryServer(t *testing.T, withHistory bool) (*httptest.Server, *history.Store) {
	t.Helper()
	ctx := context.Background()
	store, err := keystore.Open(ctx, filepath.Join(t.TempDir(), "dal.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	require.NoError(t, store.Upsert(ctx, namedKey, keyDesc))

	six := 6
	c := &collector.Collector{
		FeedHashes: map[string][]byte{testSymbol: nil, otherSymbol: nil},
		Configs:    map[string]types.Config{testSymbol: {Name: testSymbol, Decimals: &six}, otherSymbol: {Name: otherSymbol}},
	}
	server := NewServer(c, keycache.NewAPIKeyCache(time.Hour), store, hub.NewHub(nil))

	var h *history.Store
	if withHistory {
		h, err = history.Open(ctx, history.Config{Path: filepath.Join(t.TempDir(), history.FileName), Interval: 5 * time.Minute, Retention: 168 * time.Hour})
		require.NoError(t, err)
		t.Cleanup(func() { _ = h.Close() })
		server.SetHistory(h)
	}
	srv := httptest.NewServer(server)
	t.Cleanup(srv.Close)
	return srv, h
}

func getHistory(t *testing.T, url, key string) (int, map[string]history.Series) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)
	if key != "" {
		req.Header.Set("X-API-Key", key)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	var body map[string]history.Series
	if resp.StatusCode == http.StatusOK {
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	}
	return resp.StatusCode, body
}

func TestHistoryRequiresAPIKey(t *testing.T) {
	srv, _ := newHistoryServer(t, true)
	before := restCount(UnknownKeyLabel, "/history/{symbols}", "401")
	status, _ := getHistory(t, srv.URL+"/history/"+testSymbol, "")
	assert.Equal(t, http.StatusUnauthorized, status)
	status, _ = getHistory(t, srv.URL+"/history/all", "wrong")
	assert.Equal(t, http.StatusUnauthorized, status)
	assert.Equal(t, before+1, restCount(UnknownKeyLabel, "/history/{symbols}", "401"))

	okBefore := restCount(keyDesc, "/history/all", "200")
	status, _ = getHistory(t, srv.URL+"/history/all", namedKey)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, okBefore+1, restCount(keyDesc, "/history/all", "200"))
}

func TestHistoryResponse(t *testing.T) {
	srv, h := newHistoryServer(t, true)
	now := time.Now()
	hour := now.Truncate(time.Hour)
	h.Record(testSymbol, 1, "6", hour.Add(-90*time.Minute))
	h.Record(testSymbol, 2, "6", hour.Add(-65*time.Minute))
	h.Record(testSymbol, 3, "6", hour.Add(-5*time.Minute))
	h.Record(testSymbol, 4, "6", now.Add(-25*time.Hour)) // outside 24h
	require.NoError(t, h.Flush(context.Background()))

	status, body := getHistory(t, srv.URL+"/history/"+testSymbol+",eth-usdt", namedKey)
	require.Equal(t, http.StatusOK, status)
	w := func(ts time.Time) int64 { return ts.Truncate(5 * time.Minute).UnixMilli() }
	assert.Equal(t, map[string]history.Series{
		testSymbol: {Decimals: "6", Points: []history.Point{
			{Time: w(hour.Add(-90 * time.Minute)), Value: "1"},
			{Time: w(hour.Add(-65 * time.Minute)), Value: "2"},
			{Time: w(hour.Add(-5 * time.Minute)), Value: "3"},
		}},
		otherSymbol: {Decimals: collector.DefaultDecimals, Points: []history.Point{}},
	}, body)

	status, body = getHistory(t, srv.URL+"/history/all?range=7d&step=1h", namedKey)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, []history.Point{
		{Time: now.Add(-25 * time.Hour).Truncate(time.Hour).UnixMilli(), Value: "4"},
		{Time: hour.Add(-2 * time.Hour).UnixMilli(), Value: "2"},
		{Time: hour.Add(-time.Hour).UnixMilli(), Value: "3"},
	}, body[testSymbol].Points)
	assert.Contains(t, body, otherSymbol)
}

func TestHistoryBadRequests(t *testing.T) {
	srv, _ := newHistoryServer(t, true)
	for _, path := range []string{
		"/history/UNKNOWN-USDT",
		"/history/" + testSymbol + ",UNKNOWN-USDT",
		"/history/BTCUSDT",
		"/history/all?range=30d",
		"/history/all?range=1h",
		"/history/all?step=7m",
		"/history/all?step=0s",
		"/history/all?step=bogus",
		"/history/all?step=48h",
	} {
		status, _ := getHistory(t, srv.URL+path, namedKey)
		assert.Equal(t, http.StatusBadRequest, status, path)
	}
}

func TestHistoryDisabled(t *testing.T) {
	srv, _ := newHistoryServer(t, false)
	status, _ := getHistory(t, srv.URL+"/history/all", namedKey)
	assert.Equal(t, http.StatusServiceUnavailable, status)
}

func TestParseHistoryParams(t *testing.T) {
	interval, retention := 5*time.Minute, 168*time.Hour
	cases := []struct {
		query        string
		window, step time.Duration
		ok           bool
	}{
		{"", 24 * time.Hour, 5 * time.Minute, true},
		{"range=7d", 168 * time.Hour, 5 * time.Minute, true},
		{"range=24h&step=1h", 24 * time.Hour, time.Hour, true},
		{"step=10m", 24 * time.Hour, 10 * time.Minute, true},
		{"range=48h", 0, 0, false},
		{"step=1m", 0, 0, false},
	}
	for _, tc := range cases {
		q, err := url.ParseQuery(tc.query)
		require.NoError(t, err)
		window, step, err := parseHistoryParams(q, interval, retention)
		if !tc.ok {
			assert.Error(t, err, tc.query)
			continue
		}
		require.NoError(t, err, tc.query)
		assert.Equal(t, tc.window, window, tc.query)
		assert.Equal(t, tc.step, step, tc.query)
	}

	// range beyond retention
	_, _, err := parseHistoryParams(url.Values{"range": {"7d"}}, interval, 48*time.Hour)
	assert.Error(t, err)
}

func TestHistoryPointLimit(t *testing.T) {
	old := maxHistoryPoints
	maxHistoryPoints = 2 * 288 // two symbols, 24h at 5m
	t.Cleanup(func() { maxHistoryPoints = old })
	srv, _ := newHistoryServer(t, true)

	status, _ := getHistory(t, srv.URL+"/history/all", namedKey)
	assert.Equal(t, http.StatusOK, status)
	status, _ = getHistory(t, srv.URL+"/history/all?range=7d", namedKey)
	assert.Equal(t, http.StatusBadRequest, status)
	status, _ = getHistory(t, srv.URL+"/history/all?range=7d&step=1h", namedKey)
	assert.Equal(t, http.StatusOK, status)
}
