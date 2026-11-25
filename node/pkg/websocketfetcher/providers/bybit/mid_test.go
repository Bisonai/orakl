package bybit

import (
	"context"
	"encoding/json"
	"testing"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func handleRaw(t *testing.T, f *BybitFetcher, raw string) {
	t.Helper()
	var msg map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &msg))
	require.NoError(t, f.handleMessage(context.Background(), msg))
}

func TestHandleMessage_MergesTickerAndBook(t *testing.T) {
	buf := make(chan *common.FeedData, 10)
	f := &BybitFetcher{quotes: common.NewQuoteCache()}
	f.FeedMap = map[string][]int32{"KAIAUSDT": {1}}
	f.FeedDataBuffer = buf

	handleRaw(t, f, `{"topic":"orderbook.1.KAIAUSDT","ts":1795417679851,"type":"snapshot","data":{"s":"KAIAUSDT","b":[["0.0285","373"]],"a":[["0.0287","1181"]],"u":1,"seq":1},"cts":1795417679844}`)
	assert.Len(t, buf, 0, "book before ticker: no 24h volume yet")

	handleRaw(t, f, `{"topic":"tickers.KAIAUSDT","ts":1795417680000,"type":"snapshot","data":{"symbol":"KAIAUSDT","lastPrice":"0.0286","volume24h":"1000"}}`)
	require.Len(t, buf, 1)
	fd := <-buf
	assert.Equal(t, int32(1), fd.FeedID)
	assert.Equal(t, 0.0286, fd.Value, "last trade")
	assert.Equal(t, 1000.0, fd.Volume)
	assert.Equal(t, 0.0285, fd.Bid)
	assert.Equal(t, 0.0287, fd.Ask)
	require.NotNil(t, fd.BookTimestamp)
	assert.Equal(t, int64(1795417679851), fd.BookTimestamp.UnixMilli())
}
