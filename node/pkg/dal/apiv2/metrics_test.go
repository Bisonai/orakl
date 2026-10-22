package apiv2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/dal/collector"
	"bisonai.com/miko/node/pkg/dal/hub"
	"bisonai.com/miko/node/pkg/dal/utils/keycache"
	"bisonai.com/miko/node/pkg/dal/utils/keystore"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

const (
	namedKey   = "named-key"
	unnamedKey = "unnamed-key"
	keyDesc    = "metrics-test-client"
	testSymbol = "TEST-USDT"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	store, err := keystore.Open(ctx, filepath.Join(t.TempDir(), "dal.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	require.NoError(t, store.Upsert(ctx, namedKey, keyDesc))
	require.NoError(t, store.Upsert(ctx, unnamedKey, ""))

	h := hub.NewHub(map[string]struct{}{testSymbol: {}})
	go h.Start(ctx, &collector.Collector{})

	srv := httptest.NewServer(NewServer(nil, keycache.NewAPIKeyCache(time.Hour), store, h))
	t.Cleanup(srv.Close)
	return srv
}

func get(t *testing.T, url, key string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)
	if key != "" {
		req.Header.Set("X-API-Key", key)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	return resp.StatusCode
}

func restCount(key, endpoint, status string) float64 {
	return testutil.ToFloat64(restRequestsTotal.WithLabelValues(key, endpoint, status))
}

func TestEndpointLabel(t *testing.T) {
	assert.Equal(t, "/latest-data-feeds/{symbols}", endpointLabel("GET /latest-data-feeds/{symbols}"))
	assert.Equal(t, "/ws", endpointLabel("/ws"))
	assert.Equal(t, OtherEndpointLabel, endpointLabel(""))
}

func TestRESTMetricsLabels(t *testing.T) {
	srv := newTestServer(t)

	symbolsOK := restCount(keyDesc, "/symbols", "200")
	feedsBad := restCount(keyDesc, "/latest-data-feeds/{symbols}", "400")
	unauth := restCount(UnknownKeyLabel, "/latest-data-feeds/{symbols}", "401")
	unnamed := restCount(UnnamedKeyLabel, "/symbols", "200")

	// twice: second hit is served from the key cache
	assert.Equal(t, 200, get(t, srv.URL+"/symbols", namedKey))
	assert.Equal(t, 200, get(t, srv.URL+"/symbols", namedKey))
	// raw path differs, route template label is the same
	assert.Equal(t, 400, get(t, srv.URL+"/latest-data-feeds/BADSYMBOL", namedKey))
	assert.Equal(t, 401, get(t, srv.URL+"/latest-data-feeds/BTC-USDT", "wrong-key"))
	assert.Equal(t, 401, get(t, srv.URL+"/latest-data-feeds/ETH-USDT", ""))
	assert.Equal(t, 200, get(t, srv.URL+"/symbols", unnamedKey))

	assert.Equal(t, symbolsOK+2, restCount(keyDesc, "/symbols", "200"))
	assert.Equal(t, feedsBad+1, restCount(keyDesc, "/latest-data-feeds/{symbols}", "400"))
	assert.Equal(t, unauth+2, restCount(UnknownKeyLabel, "/latest-data-feeds/{symbols}", "401"))
	assert.Equal(t, unnamed+1, restCount(UnnamedKeyLabel, "/symbols", "200"))

	// raw key never used as a label value
	assert.Zero(t, restCount(namedKey, "/symbols", "200"))
}

func TestHealthAndMetricsNotCounted(t *testing.T) {
	srv := newTestServer(t)

	before := testutil.CollectAndCount(restRequestsTotal)
	assert.Equal(t, 200, get(t, srv.URL+"/", ""))
	assert.Equal(t, 200, get(t, srv.URL+"/metrics", ""))
	assert.Equal(t, before, testutil.CollectAndCount(restRequestsTotal))
}

func TestWebsocketMetrics(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.Background()

	conns := testutil.ToFloat64(wsConnectionsTotal.WithLabelValues(keyDesc))
	subs := testutil.ToFloat64(wsSubscriptionsTotal.WithLabelValues(keyDesc))

	c, _, err := websocket.Dial(ctx, "ws"+srv.URL[len("http"):]+"/ws", &websocket.DialOptions{
		HTTPHeader: http.Header{"X-API-Key": []string{namedKey}},
	})
	require.NoError(t, err)

	require.NoError(t, wsjson.Write(ctx, c, hub.Subscription{
		Method: "SUBSCRIBE",
		Params: []string{"submission@" + testSymbol, "submission@UNKNOWN-USDT"},
	}))

	assert.Eventually(t, func() bool {
		return testutil.ToFloat64(wsSubscriptionsTotal.WithLabelValues(keyDesc)) == subs+1
	}, 2*time.Second, 10*time.Millisecond, "only the valid subscription is counted")
	assert.Equal(t, conns+1, testutil.ToFloat64(wsConnectionsTotal.WithLabelValues(keyDesc)))
	assert.Equal(t, float64(1), testutil.ToFloat64(wsActiveConnections.WithLabelValues(keyDesc)))

	require.NoError(t, c.Close(websocket.StatusNormalClosure, ""))
	assert.Eventually(t, func() bool {
		return testutil.ToFloat64(wsActiveConnections.WithLabelValues(keyDesc)) == 0
	}, 2*time.Second, 10*time.Millisecond)
}

func TestWebsocketUnauthorized(t *testing.T) {
	srv := newTestServer(t)
	_, resp, err := websocket.Dial(context.Background(), "ws"+srv.URL[len("http"):]+"/ws", &websocket.DialOptions{
		HTTPHeader: http.Header{"X-API-Key": []string{"wrong-key"}},
	})
	assert.Error(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
