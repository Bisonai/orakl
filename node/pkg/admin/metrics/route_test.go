package metrics

import (
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/common/types"
	nodemetrics "bisonai.com/miko/node/pkg/metrics"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsRoute(t *testing.T) {
	nodemetrics.SetLocalAggregateConfigs([]string{"ROUTE-TEST"})
	nodemetrics.SetGlobalAggregateConfigs([]string{"ROUTE-TEST"})
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

func TestMetricsRouteFeeds(t *testing.T) {
	ts := time.Unix(1700000002, 0)
	data := &types.LatestFeedDataMap{FeedDataMap: map[int32]*types.FeedData{
		1: {FeedID: 1, Value: 0.0065, Volume: 12, Timestamp: &ts},
	}}
	nodemetrics.SetFeeds(data, map[int32]nodemetrics.FeedLabels{1: {Config: "ROUTE-TEST", Feed: "Binance-ROUTE-TEST"}})
	t.Cleanup(func() { nodemetrics.SetFeeds(nil, nil) })

	app := fiber.New()
	Routes(app.Group("/api/v1"))

	resp, err := app.Test(httptest.NewRequest("GET", "/api/v1/metrics", nil))
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	text := string(body)
	assert.Contains(t, text, `orakl_node_feed_value{config="ROUTE-TEST",feed="Binance-ROUTE-TEST"} 0.0065`)
	assert.Contains(t, text, `orakl_node_feed_timestamp_seconds{config="ROUTE-TEST",feed="Binance-ROUTE-TEST"} 1.700000002e+09`)
	assert.Contains(t, text, `orakl_node_feed_volume{config="ROUTE-TEST",feed="Binance-ROUTE-TEST"} 12`)
}
