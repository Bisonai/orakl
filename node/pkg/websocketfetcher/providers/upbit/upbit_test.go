package upbit

import (
	"context"
	"encoding/json"
	"testing"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestFetcher() (*UpbitFetcher, chan *common.FeedData) {
	buf := make(chan *common.FeedData, 10)
	f := &UpbitFetcher{player: "Upbit", quotes: common.NewQuoteCache()}
	f.FeedMap = map[string][]int32{"WAVES-KRW": {1}}
	f.FeedDataBuffer = buf
	return f, buf
}

func handle(t *testing.T, f *UpbitFetcher, raw string) {
	t.Helper()
	var msg map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &msg))
	require.NoError(t, f.handleMessage(context.Background(), msg))
}

const (
	tickerMsg = `{"ty":"ticker","cd":"KRW-WAVES","tp":214,"atv24h":4465183.1,"ttms":1795396000000,"tms":1795396001000,"st":"SNAPSHOT"}`
	bookMsg   = `{"ty":"orderbook","cd":"KRW-WAVES","tms":1795396941335,"obu":[{"ap":215,"bp":214,"as":1,"bs":1}],"st":"REALTIME"}`
)

func TestHandleMessage_MergesTickerAndBook(t *testing.T) {
	f, buf := newTestFetcher()

	// book before any ticker: nothing to emit (no 24h volume yet)
	handle(t, f, bookMsg)
	assert.Len(t, buf, 0)

	handle(t, f, tickerMsg)
	require.Len(t, buf, 1)
	fd := <-buf
	assert.Equal(t, int32(1), fd.FeedID)
	assert.Equal(t, 214.0, fd.Value)
	assert.Equal(t, 4465183.1, fd.Volume)
	assert.Equal(t, int64(1795396000000), fd.Timestamp.UnixMilli(), "trade time")
	assert.Equal(t, 214.0, fd.Bid)
	assert.Equal(t, 215.0, fd.Ask)
	assert.Equal(t, int64(1795396941335), fd.BookTimestamp.UnixMilli())

	handle(t, f, `{"ty":"orderbook","cd":"KRW-WAVES","tms":1795396950000,"obu":[{"ap":216,"bp":215,"as":1,"bs":1}]}`)
	require.Len(t, buf, 1)
	fd = <-buf
	assert.Equal(t, 214.0, fd.Value, "last trade unchanged by book updates")
	assert.Equal(t, 215.0, fd.Bid)
	assert.Equal(t, 216.0, fd.Ask)
	assert.Equal(t, int64(1795396950000), fd.BookTimestamp.UnixMilli())
}

func TestHandleMessage_UnknownMarketBook(t *testing.T) {
	f, buf := newTestFetcher()
	var msg map[string]any
	require.NoError(t, json.Unmarshal([]byte(`{"ty":"orderbook","cd":"KRW-BTC","tms":1,"obu":[{"ap":2,"bp":1}]}`), &msg))
	assert.Error(t, f.handleMessage(context.Background(), msg))
	assert.Len(t, buf, 0)
}
