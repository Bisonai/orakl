package binance

import (
	"context"
	"encoding/json"
	"testing"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func handleRaw(t *testing.T, f *BinanceFetcher, raw string) {
	t.Helper()
	var msg map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &msg))
	require.NoError(t, f.handleMessage(context.Background(), msg))
}

func TestHandleMessage_MergesTickerAndBook(t *testing.T) {
	buf := make(chan *common.FeedData, 10)
	f := &BinanceFetcher{quotes: common.NewQuoteCache()}
	f.FeedMap = map[string][]int32{"KAIAUSDT": {1}}
	f.FeedDataBuffer = buf

	handleRaw(t, f, `{"stream":"kaiausdt@depth5@1000ms","data":{"lastUpdateId":1,"bids":[["0.0285","100"],["0.0284","1"]],"asks":[["0.0287","50"],["0.0288","1"]]}}`)
	assert.Len(t, buf, 0, "book before ticker: no 24h volume yet")

	handleRaw(t, f, `{"stream":"kaiausdt@miniTicker","data":{"e":"24hrMiniTicker","E":1795417681000,"s":"KAIAUSDT","c":"0.0286","o":"0.028","h":"0.029","l":"0.027","v":"1000","q":"28.6"}}`)
	require.Len(t, buf, 1)
	fd := <-buf
	assert.Equal(t, int32(1), fd.FeedID)
	assert.Equal(t, 0.0286, fd.Value, "last trade")
	assert.Equal(t, 1000.0, fd.Volume)
	assert.Equal(t, 0.0285, fd.Bid)
	assert.Equal(t, 0.0287, fd.Ask)
	require.NotNil(t, fd.BookTimestamp)
	// binance depth has no timestamp: receive time
}
