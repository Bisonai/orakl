package metrics

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	nodemetrics "bisonai.com/miko/node/pkg/metrics"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func familyNames(t *testing.T, body io.Reader) []string {
	t.Helper()
	parser := expfmt.NewTextParser(model.UTF8Validation)
	families, err := parser.TextToMetricFamilies(body)
	require.NoError(t, err)
	names := make([]string, 0, len(families))
	for name := range families {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func TestMetricsMuxMatchesAdminRoute(t *testing.T) {
	nodemetrics.SetLocalAggregateConfigs([]string{"MUX-TEST"})
	nodemetrics.SetLocalAggregate("MUX-TEST", 100, time.Unix(1700000000, 0))
	t.Cleanup(func() { nodemetrics.SetLocalAggregateConfigs(nil) })

	app := fiber.New()
	Routes(app.Group("/api/v1"))
	adminResp, err := app.Test(httptest.NewRequest("GET", "/api/v1/metrics", nil))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, adminResp.StatusCode)
	adminNames := familyNames(t, adminResp.Body)

	rec := httptest.NewRecorder()
	NewMux().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	muxNames := familyNames(t, rec.Body)

	assert.Equal(t, adminNames, muxNames)
	assert.Contains(t, muxNames, "orakl_node_local_aggregate_value")
}

func TestMetricsMuxOtherPaths404(t *testing.T) {
	mux := NewMux()
	for _, path := range []string{"/", "/api/v1/metrics", "/metrics/", "/api/v1/fetcher/stop"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		assert.Equal(t, http.StatusNotFound, rec.Code, path)
	}
}

func TestServe(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := Serve(ln)
	t.Cleanup(func() { srv.Close() })
	base := "http://" + ln.Addr().String()

	resp, err := http.Get(base + "/metrics")
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), "go_goroutines")

	resp, err = http.Get(base + "/api/v1/metrics")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestStartServerBindFailure(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })

	_, err = StartServer(fmt.Sprint(ln.Addr().(*net.TCPAddr).Port))
	assert.Error(t, err)
}

func TestStartServerInvalidPort(t *testing.T) {
	for _, port := range []string{"0", "abc", "0.0.0.0:9100", "70000"} {
		srv, err := StartServer(port)
		if !assert.Error(t, err, port) {
			srv.Close()
		}
	}
}
