package metrics

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/common/types"
	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newFeedDataMap(data ...*types.FeedData) *types.LatestFeedDataMap {
	m := &types.LatestFeedDataMap{FeedDataMap: make(map[int32]*types.FeedData)}
	for _, d := range data {
		m.FeedDataMap[d.FeedID] = d
	}
	return m
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestFeedCollectorEmitsLoadedFeeds(t *testing.T) {
	c := &feedCollector{}
	data := newFeedDataMap(
		&types.FeedData{FeedID: 1, Value: 1.5, Volume: 1000, Timestamp: ptrTime(time.Unix(1700000000, 500_000_000))},
		&types.FeedData{FeedID: 2, Value: 0.0065, Timestamp: ptrTime(time.Unix(1700000001, 0))},
		&types.FeedData{FeedID: 3, Value: 9, Timestamp: ptrTime(time.Unix(1700000002, 0))}, // not loaded
	)
	c.setFeeds(data, map[int32]FeedLabels{
		1: {Config: "BTC-USDT", Feed: "Binance-BTC-USDT"},
		2: {Config: "JPYC-USDT", Feed: "UniswapV4-Ethereum-USDC-JPYC"},
		4: {Config: "BTC-USDT", Feed: "Coinbase-BTC-USDT"}, // loaded, no data yet
	})

	expected := `
# HELP orakl_node_feed_timestamp_seconds Unix timestamp of the latest value per feed
# TYPE orakl_node_feed_timestamp_seconds gauge
orakl_node_feed_timestamp_seconds{config="BTC-USDT",feed="Binance-BTC-USDT"} 1.7000000005e+09
orakl_node_feed_timestamp_seconds{config="JPYC-USDT",feed="UniswapV4-Ethereum-USDC-JPYC"} 1.700000001e+09
# HELP orakl_node_feed_value Latest raw value per feed
# TYPE orakl_node_feed_value gauge
orakl_node_feed_value{config="BTC-USDT",feed="Binance-BTC-USDT"} 1.5
orakl_node_feed_value{config="JPYC-USDT",feed="UniswapV4-Ethereum-USDC-JPYC"} 0.0065
# HELP orakl_node_feed_volume Latest volume per feed (only feeds reporting volume)
# TYPE orakl_node_feed_volume gauge
orakl_node_feed_volume{config="BTC-USDT",feed="Binance-BTC-USDT"} 1000
`
	require.NoError(t, testutil.CollectAndCompare(c, strings.NewReader(expected)))
}

func TestFeedCollectorDropsRemovedFeeds(t *testing.T) {
	c := &feedCollector{}
	data := newFeedDataMap(
		&types.FeedData{FeedID: 1, Value: 1, Timestamp: ptrTime(time.Unix(100, 0))},
		&types.FeedData{FeedID: 2, Value: 2, Timestamp: ptrTime(time.Unix(200, 0))},
	)
	c.setFeeds(data, map[int32]FeedLabels{
		1: {Config: "A", Feed: "KEEP"},
		2: {Config: "A", Feed: "REMOVED"},
	})
	assert.Equal(t, 4, testutil.CollectAndCount(c))

	// refresh without feed 2: its data is still in the map but not emitted
	c.setFeeds(data, map[int32]FeedLabels{1: {Config: "A", Feed: "KEEP"}})
	assert.Equal(t, 1, testutil.CollectAndCount(c, metricnames.FeedValue))
	assert.Equal(t, 1, testutil.CollectAndCount(c, metricnames.FeedTimestamp))

	c.setFeeds(data, nil)
	assert.Equal(t, 0, testutil.CollectAndCount(c))
}

func TestFeedCollectorSkipsMissingTimestamp(t *testing.T) {
	c := &feedCollector{}
	c.setFeeds(newFeedDataMap(&types.FeedData{FeedID: 1, Value: 3}), map[int32]FeedLabels{1: {Config: "A", Feed: "F"}})
	assert.Equal(t, 1, testutil.CollectAndCount(c, metricnames.FeedValue))
	assert.Equal(t, 0, testutil.CollectAndCount(c, metricnames.FeedTimestamp))
}

// scrapes race with map writes and set swaps; run with -race
func TestFeedCollectorConcurrentScrape(t *testing.T) {
	c := &feedCollector{}
	reg := prometheus.NewPedanticRegistry()
	require.NoError(t, reg.Register(c))

	data := newFeedDataMap()
	names := map[int32]FeedLabels{}
	for i := int32(1); i <= 100; i++ {
		names[i] = FeedLabels{Config: "C", Feed: fmt.Sprintf("F%d", i)}
	}
	c.setFeeds(data, names)

	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			now := time.Now()
			_ = data.SetLatestFeedData([]*types.FeedData{{FeedID: int32(i%100 + 1), Value: float64(i), Volume: 1, Timestamp: &now}})
		}
	}()
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			c.setFeeds(data, names)
		}
	}()

	for i := 0; i < 200; i++ {
		_, err := reg.Gather()
		require.NoError(t, err)
	}
	close(stop)
	wg.Wait()
}

func TestSetFeedsServedByDefaultRegistry(t *testing.T) {
	t.Cleanup(func() { SetFeeds(nil, nil) })
	SetFeeds(newFeedDataMap(&types.FeedData{FeedID: 1, Value: 7, Timestamp: ptrTime(time.Unix(100, 0))}),
		map[int32]FeedLabels{1: {Config: "DEFAULT", Feed: "F"}})

	mfs, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	found := false
	for _, mf := range mfs {
		if mf.GetName() == metricnames.FeedValue {
			found = len(mf.GetMetric()) == 1
		}
	}
	assert.True(t, found)
}
