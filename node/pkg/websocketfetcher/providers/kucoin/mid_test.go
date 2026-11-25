package kucoin

import (
	"context"
	"encoding/json"
	"testing"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func handleRaw(t *testing.T, f *KucoinFetcher, raw string) {
	t.Helper()
	var msg map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &msg))
	require.NoError(t, f.handleMessage(context.Background(), msg))
}

func TestHandleMessage_MergesTickerAndBook(t *testing.T) {
	buf := make(chan *common.FeedData, 10)
	f := &KucoinFetcher{quotes: common.NewQuoteCache()}
	f.FeedMap = map[string][]int32{"KAIA-USDT": {1}}
	f.FeedDataBuffer = buf

	handleRaw(t, f, `{"topic":"/spotMarket/level1:KAIA-USDT","type":"message","subject":"level1","data":{"asks":["0.0287","1"],"bids":["0.0285","1"],"timestamp":1795417715330}}`)
	assert.Len(t, buf, 0, "book before ticker: no 24h volume yet")

	handleRaw(t, f, `{"type":"message","topic":"/market/snapshot:KAIA-USDT","subject":"trade.snapshot","data":{"sequence":"1","data":{"symbol":"KAIA-USDT","lastTradedPrice":0.0286,"vol":1000,"datetime":1795417715000}}}`)
	require.Len(t, buf, 1)
	fd := <-buf
	assert.Equal(t, int32(1), fd.FeedID)
	assert.Equal(t, 0.0286, fd.Value, "last trade")
	assert.Equal(t, 1000.0, fd.Volume)
	assert.Equal(t, 0.0285, fd.Bid)
	assert.Equal(t, 0.0287, fd.Ask)
	require.NotNil(t, fd.BookTimestamp)
	assert.Equal(t, int64(1795417715330), fd.BookTimestamp.UnixMilli())
}
