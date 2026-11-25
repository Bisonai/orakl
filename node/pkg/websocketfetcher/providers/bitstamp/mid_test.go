package bitstamp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func toMsg(t *testing.T, raw string) map[string]any {
	t.Helper()
	var msg map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &msg))
	return msg
}

func TestHandleMessage_MergesTradesAndOrderBook(t *testing.T) {
	buf := make(chan *common.FeedData, 10)
	f := &BitstampFetcher{quotes: common.NewQuoteCache()}
	f.FeedMap = map[string][]int32{"BTCUSDT": {1}}
	f.FeedDataBuffer = buf
	f.VolumeCacheMap = common.VolumeCacheMap{Map: map[int32]common.VolumeCache{}}
	f.VolumeCacheMap.Set(1, common.VolumeCache{UpdatedAt: time.Now(), Volume: 7})
	ctx := context.Background()

	book := `{"data":{"timestamp":"1795500421","microtimestamp":"1795500421573722","bids":[["60474.87","0.08277984"],["60470.04","0.00827544"]],"asks":[["60490.10","0.1"],["60495.00","0.2"]]},"channel":"order_book_btcusdt","event":"data"}`
	require.NoError(t, f.handleMessage(ctx, toMsg(t, book)))
	assert.Len(t, buf, 0, "book before first trade")

	trade := `{"data":{"id":680786402,"timestamp":"1795501215","amount":0.00102674,"amount_str":"0.00102674","price":60480.5,"price_str":"60480.5","type":0,"microtimestamp":"1795501215987000","buy_order_id":2075304923852800,"sell_order_id":2075304923852801},"channel":"live_trades_btcusdt","event":"trade"}`
	require.NoError(t, f.handleMessage(ctx, toMsg(t, trade)))
	require.Len(t, buf, 1)
	fd := <-buf
	assert.Equal(t, 60480.5, fd.Value)
	assert.Equal(t, 7.0, fd.Volume)
	assert.Equal(t, 60474.87, fd.Bid)
	assert.Equal(t, 60490.10, fd.Ask)
	require.NotNil(t, fd.BookTimestamp)
	assert.Equal(t, int64(1795500421573722), fd.BookTimestamp.UnixMicro())

	ack := `{"event":"bts:subscription_succeeded","channel":"order_book_btcusdt","data":{}}`
	require.NoError(t, f.handleMessage(ctx, toMsg(t, ack)))
	assert.Len(t, buf, 0)
}

func TestHandleMessage_BookOnlyBeforeFirstTrade(t *testing.T) {
	buf := make(chan *common.FeedData, 10)
	fi, err := New(context.Background(),
		common.WithFeedMaps(common.FeedMaps{Combined: map[string][]int32{"BTCUSDT": {1}}}),
		common.WithFeedDataBuffer(buf))
	require.NoError(t, err)
	f := fi.(*BitstampFetcher)
	ctx := context.Background()

	book := `{"data":{"timestamp":"1795500421","microtimestamp":"1795500421573722","bids":[["60474.87","0.08277984"]],"asks":[["60490.13","0.1"]]},"channel":"order_book_btcusdt","event":"data"}`
	require.NoError(t, f.handleMessage(ctx, toMsg(t, book)))
	assert.Len(t, buf, 0, "24h volume not known yet")

	f.VolumeCacheMap.Set(1, common.VolumeCache{UpdatedAt: time.Now(), Volume: 7})
	require.NoError(t, f.handleMessage(ctx, toMsg(t, book)))
	require.Len(t, buf, 1)
	fd := <-buf
	assert.True(t, fd.BookOnly)
	assert.InDelta(t, 60482.5, fd.Value, 1e-9, "mid")
	assert.Equal(t, 7.0, fd.Volume)
	assert.Equal(t, int64(1795500421573722), fd.Timestamp.UnixMicro(), "book time")

	trade := `{"data":{"id":680786402,"timestamp":"1795501215","amount":0.00102674,"amount_str":"0.00102674","price":60480.5,"price_str":"60480.5","type":0,"microtimestamp":"1795501215987000","buy_order_id":2075304923852800,"sell_order_id":2075304923852801},"channel":"live_trades_btcusdt","event":"trade"}`
	require.NoError(t, f.handleMessage(ctx, toMsg(t, trade)))
	require.Len(t, buf, 1)
	fd = <-buf
	assert.False(t, fd.BookOnly)
	assert.Equal(t, 60480.5, fd.Value)
	assert.Equal(t, 60474.87, fd.Bid)
}
