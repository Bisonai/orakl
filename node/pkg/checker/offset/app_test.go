package offset

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	nodemetrics "bisonai.com/miko/node/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetAlarmCounts() {
	localAggregateAlarmCount = make(map[string]int)
	globalAggregateAlarmCount = make(map[string]int)
}

func metricsText(now time.Time, localDelay, globalDelay time.Duration, localValue, globalValue float64) string {
	nowSeconds := float64(now.Unix())
	return fmt.Sprintf(`# TYPE orakl_node_local_aggregate_value gauge
orakl_node_local_aggregate_value{name="BTC-USDT"} %v
# TYPE orakl_node_local_aggregate_timestamp_seconds gauge
orakl_node_local_aggregate_timestamp_seconds{name="BTC-USDT"} %v
# TYPE orakl_node_global_aggregate_value gauge
orakl_node_global_aggregate_value{name="BTC-USDT"} %v
# TYPE orakl_node_global_aggregate_round gauge
orakl_node_global_aggregate_round{name="BTC-USDT"} 10
# TYPE orakl_node_global_aggregate_timestamp_seconds gauge
orakl_node_global_aggregate_timestamp_seconds{name="BTC-USDT"} %v
`, localValue, nowSeconds-localDelay.Seconds(), globalValue, nowSeconds-globalDelay.Seconds())
}

func mustParse(t *testing.T, text string) map[string]*aggregateMetrics {
	t.Helper()
	aggregates, err := parseAggregateMetrics(strings.NewReader(text))
	require.NoError(t, err)
	return aggregates
}

func TestParseAggregateMetrics(t *testing.T) {
	now := time.Unix(1700000000, 0)
	aggregates := mustParse(t, metricsText(now, time.Second, 2*time.Second, 100, 101)+`# TYPE go_goroutines gauge
go_goroutines 10
`)

	require.Contains(t, aggregates, "BTC-USDT")
	btc := aggregates["BTC-USDT"]
	assert.Equal(t, 100.0, *btc.LocalValue)
	assert.Equal(t, 1699999999.0, *btc.LocalTimestamp)
	assert.Equal(t, 101.0, *btc.GlobalValue)
	assert.Equal(t, 1699999998.0, *btc.GlobalTimestamp)
}

func TestParseAggregateMetricsPartial(t *testing.T) {
	aggregates, err := parseAggregateMetrics(strings.NewReader(`# TYPE orakl_node_global_aggregate_timestamp_seconds gauge
orakl_node_global_aggregate_timestamp_seconds{name="NEW-USDT"} 1.7e+09
`))
	require.NoError(t, err)
	require.Contains(t, aggregates, "NEW-USDT")
	assert.Nil(t, aggregates["NEW-USDT"].LocalValue)
	assert.Nil(t, aggregates["NEW-USDT"].LocalTimestamp)
	assert.Nil(t, aggregates["NEW-USDT"].GlobalValue)
	assert.Equal(t, 1.7e9, *aggregates["NEW-USDT"].GlobalTimestamp)
}

func TestOffsetAlarmNoAlarm(t *testing.T) {
	resetAlarmCounts()
	now := time.Unix(1700000000, 0)
	aggregates := mustParse(t, metricsText(now, time.Second, time.Second, 100, 101))

	for i := 0; i < AlarmOffsetPerPair+2; i++ {
		assert.Empty(t, offsetAlarmMessage(aggregates, now))
	}
}

func TestOffsetAlarmLocalDelay(t *testing.T) {
	resetAlarmCounts()
	now := time.Unix(1700000000, 0)
	aggregates := mustParse(t, metricsText(now, time.Minute, time.Second, 100, 100))

	// up to AlarmOffsetInTotal: silent
	for i := 0; i < AlarmOffsetInTotal; i++ {
		assert.Empty(t, offsetAlarmMessage(aggregates, now))
	}
	// beyond AlarmOffsetInTotal: summary count only
	for i := AlarmOffsetInTotal; i < AlarmOffsetPerPair; i++ {
		assert.Equal(t, "1 local aggregate offsets delayed\n", offsetAlarmMessage(aggregates, now))
	}
	// beyond AlarmOffsetPerPair: per-pair alarm, counter reset
	assert.Equal(t, "(local aggregate offset delayed) BTC-USDT: 60 seconds\n", offsetAlarmMessage(aggregates, now))
	assert.Equal(t, 0, localAggregateAlarmCount["BTC-USDT"])
	assert.Equal(t, 0, globalAggregateAlarmCount["BTC-USDT"])
}

func TestOffsetAlarmGlobalDelay(t *testing.T) {
	resetAlarmCounts()
	now := time.Unix(1700000000, 0)
	aggregates := mustParse(t, metricsText(now, time.Second, time.Minute, 100, 100))

	var msg string
	for i := 0; i <= AlarmOffsetPerPair; i++ {
		msg = offsetAlarmMessage(aggregates, now)
	}
	assert.Equal(t, "(global aggregate offset delayed) BTC-USDT: 60 seconds\n", msg)
	assert.Equal(t, 0, localAggregateAlarmCount["BTC-USDT"])
}

func TestOffsetAlarmDelayRecoveryResetsCount(t *testing.T) {
	resetAlarmCounts()
	now := time.Unix(1700000000, 0)
	delayed := mustParse(t, metricsText(now, time.Minute, time.Minute, 100, 100))
	fresh := mustParse(t, metricsText(now, time.Second, time.Second, 100, 100))

	offsetAlarmMessage(delayed, now)
	assert.Equal(t, 1, localAggregateAlarmCount["BTC-USDT"])
	assert.Equal(t, 1, globalAggregateAlarmCount["BTC-USDT"])

	offsetAlarmMessage(fresh, now)
	assert.Equal(t, 0, localAggregateAlarmCount["BTC-USDT"])
	assert.Equal(t, 0, globalAggregateAlarmCount["BTC-USDT"])
}

func TestOffsetAlarmPriceDifference(t *testing.T) {
	resetAlarmCounts()
	now := time.Unix(1700000000, 0)
	aggregates := mustParse(t, metricsText(now, time.Second, time.Second, 100, 120))

	assert.Equal(t, "(latest aggregate price difference) BTC-USDT: -20 (globalAggregate: 120, localAggregate: 100)\n", offsetAlarmMessage(aggregates, now))
}

func TestOffsetAlarmSkipsMissingValuesAndDelisted(t *testing.T) {
	resetAlarmCounts()
	now := time.Unix(1700000000, 0)
	aggregates, err := parseAggregateMetrics(strings.NewReader(`# TYPE orakl_node_local_aggregate_value gauge
orakl_node_local_aggregate_value{name="ONLY-LOCAL"} 100
orakl_node_local_aggregate_value{name="JOY-USDT"} 100
# TYPE orakl_node_local_aggregate_timestamp_seconds gauge
orakl_node_local_aggregate_timestamp_seconds{name="ONLY-LOCAL"} 1.7e+09
orakl_node_local_aggregate_timestamp_seconds{name="JOY-USDT"} 1
# TYPE orakl_node_global_aggregate_value gauge
orakl_node_global_aggregate_value{name="JOY-USDT"} 1
`))
	require.NoError(t, err)

	for i := 0; i <= AlarmOffsetPerPair; i++ {
		assert.Empty(t, offsetAlarmMessage(aggregates, now))
	}
}

func TestFetchAggregateMetrics(t *testing.T) {
	now := time.Unix(1700000000, 0)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/metrics" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(metricsText(now, time.Second, time.Second, 100, 100)))
	}))
	defer srv.Close()

	aggregates, err := fetchAggregateMetrics(context.Background(), srv.URL+"/api/v1"+nodeMetricsEndpoint)
	require.NoError(t, err)
	require.Contains(t, aggregates, "BTC-USDT")
	assert.Equal(t, 100.0, *aggregates["BTC-USDT"].LocalValue)

	_, err = fetchAggregateMetrics(context.Background(), srv.URL+"/missing")
	assert.Error(t, err)
}

// reads what the node actually serves (pkg/metrics via promhttp)
func TestFetchAggregateMetricsFromNodeRegistry(t *testing.T) {
	nodemetrics.SetLocalAggregate("REGISTRY-USDT", 200, time.Unix(1700000000, 0))
	nodemetrics.SetGlobalAggregate("REGISTRY-USDT", 201, 3, time.Unix(1700000001, 0))
	t.Cleanup(func() {
		nodemetrics.SetLocalAggregateConfigs(nil)
		nodemetrics.SetGlobalAggregateConfigs(nil)
	})

	srv := httptest.NewServer(promhttp.Handler())
	defer srv.Close()

	aggregates, err := fetchAggregateMetrics(context.Background(), srv.URL)
	require.NoError(t, err)
	require.Contains(t, aggregates, "REGISTRY-USDT")
	got := aggregates["REGISTRY-USDT"]
	assert.Equal(t, 200.0, *got.LocalValue)
	assert.Equal(t, 1700000000.0, *got.LocalTimestamp)
	assert.Equal(t, 201.0, *got.GlobalValue)
	assert.Equal(t, 1700000001.0, *got.GlobalTimestamp)
}
