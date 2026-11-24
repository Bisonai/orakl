package btse

import (
	"context"
	"encoding/json"
	"testing"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func handleRaw(t *testing.T, f *BtseFetcher, raw string) {
	t.Helper()
	var msg map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &msg))
	require.NoError(t, f.handleMessage(context.Background(), msg))
}

func TestHandleMessage_MergesTradeAndSnapshotL1(t *testing.T) {
	buf := make(chan *common.FeedData, 10)
	f := &BtseFetcher{quotes: common.NewQuoteCache()}
	f.FeedMap = map[string][]int32{"DOT-USDT": {1}}
	f.FeedDataBuffer = buf
	f.VolumeCacheMap = common.VolumeCacheMap{Map: map[int32]common.VolumeCache{}}

	// snapshotL1 data is an object (trade history data is an array)
	handleRaw(t, f, `{"topic":"snapshotL1:DOT-USDT_0","data":{"bids":[["0.676","4502.70"]],"asks":[["0.679","3252.72"]],"type":"snapshotL1","symbol":"DOT-USDT","timestamp":1795487651723}}`)
	assert.Len(t, buf, 0, "book before trade")

	handleRaw(t, f, `{"topic":"tradeHistoryApi:DOT-USDT","data":[{"symbol":"DOT-USDT","side":"BUY","size":10,"price":0.678,"tradeId":1,"timestamp":1795487650000}]}`)
	require.Len(t, buf, 1)
	fd := <-buf
	assert.Equal(t, 0.678, fd.Value)
	assert.Equal(t, 0.676, fd.Bid)
	assert.Equal(t, 0.679, fd.Ask)
	assert.Equal(t, int64(1795487651723), fd.BookTimestamp.UnixMilli())
}
