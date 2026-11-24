package gateio

import (
	"context"
	"encoding/json"
	"testing"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func handleRaw(t *testing.T, f *GateioFetcher, raw string) {
	t.Helper()
	var msg map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &msg))
	require.NoError(t, f.handleMessage(context.Background(), msg))
}

func TestHandleMessage_MergesTickerAndBook(t *testing.T) {
	buf := make(chan *common.FeedData, 10)
	f := &GateioFetcher{quotes: common.NewQuoteCache()}
	f.FeedMap = map[string][]int32{"KAIA-USDT": {1}}
	f.FeedDataBuffer = buf

	handleRaw(t, f, `{"time":1795417681,"time_ms":1795417681082,"channel":"spot.book_ticker","event":"update","result":{"t":1795417681077,"u":1,"s":"KAIA_USDT","b":"0.0285","B":"1","a":"0.0287","A":"1"}}`)
	assert.Len(t, buf, 0, "book before ticker: no 24h volume yet")

	handleRaw(t, f, `{"time":1795417681,"time_ms":1795417681068,"channel":"spot.tickers","event":"update","result":{"currency_pair":"KAIA_USDT","last":"0.0286","base_volume":"1000"}}`)
	require.Len(t, buf, 1)
	fd := <-buf
	assert.Equal(t, int32(1), fd.FeedID)
	assert.Equal(t, 0.0286, fd.Value, "last trade")
	assert.Equal(t, 1000.0, fd.Volume)
	assert.Equal(t, 0.0285, fd.Bid)
	assert.Equal(t, 0.0287, fd.Ask)
	require.NotNil(t, fd.BookTimestamp)
	assert.Equal(t, int64(1795417681077), fd.BookTimestamp.UnixMilli())
}
