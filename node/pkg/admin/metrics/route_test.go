package metrics

import (
	"io"
	"net/http/httptest"
	"testing"
	"time"

	nodemetrics "bisonai.com/miko/node/pkg/metrics"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsRoute(t *testing.T) {
	nodemetrics.SetLocalAggregate("ROUTE-TEST", 100, time.Unix(1700000000, 0))
	nodemetrics.SetGlobalAggregate("ROUTE-TEST", 101, 7, time.Unix(1700000001, 0))
	t.Cleanup(func() {
		nodemetrics.SetLocalAggregateConfigs(nil)
		nodemetrics.SetGlobalAggregateConfigs(nil)
	})

	app := fiber.New()
	Routes(app.Group("/api/v1"))

	resp, err := app.Test(httptest.NewRequest("GET", "/api/v1/metrics", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	text := string(body)
	assert.Contains(t, text, `orakl_node_local_aggregate_value{name="ROUTE-TEST"} 100`)
	assert.Contains(t, text, `orakl_node_local_aggregate_timestamp_seconds{name="ROUTE-TEST"} 1.7e+09`)
	assert.Contains(t, text, `orakl_node_global_aggregate_value{name="ROUTE-TEST"} 101`)
	assert.Contains(t, text, `orakl_node_global_aggregate_round{name="ROUTE-TEST"} 7`)
	assert.Contains(t, text, `orakl_node_global_aggregate_timestamp_seconds{name="ROUTE-TEST"} 1.700000001e+09`)
}
