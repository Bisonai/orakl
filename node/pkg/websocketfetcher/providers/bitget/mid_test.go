package bitget

import (
	"context"
	"encoding/json"
	"testing"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func handleRaw(t *testing.T, f *BitgetFetcher, raw string) {
	t.Helper()
	var msg map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &msg))
	require.NoError(t, f.handleMessage(context.Background(), msg))
}

func TestHandleMessage_MergesTickerAndBook(t *testing.T) {
	buf := make(chan *common.FeedData, 10)
	f := &BitgetFetcher{quotes: common.NewQuoteCache()}
	f.FeedMap = map[string][]int32{"KAIAUSDT": {1}}
	f.FeedDataBuffer = buf

	handleRaw(t, f, `{"action":"snapshot","arg":{"instType":"SPOT","channel":"books1","instId":"KAIAUSDT"},"data":[{"asks":[["0.0287","50"]],"bids":[["0.0285","100"]],"ts":"1795417681154","seq":1,"pseq":0}],"ts":1795417681155}`)
	assert.Len(t, buf, 0, "book before ticker: no 24h volume yet")

	handleRaw(t, f, `{"action":"snapshot","arg":{"instType":"SPOT","channel":"ticker","instId":"KAIAUSDT"},"data":[{"instId":"KAIAUSDT","lastPr":"0.0286","baseVolume":"1000","ts":"1795417681000"}]}`)
	require.Len(t, buf, 1)
	fd := <-buf
	assert.Equal(t, int32(1), fd.FeedID)
	assert.Equal(t, 0.0286, fd.Value, "last trade")
	assert.Equal(t, 1000.0, fd.Volume)
	assert.Equal(t, 0.0285, fd.Bid)
	assert.Equal(t, 0.0287, fd.Ask)
	require.NotNil(t, fd.BookTimestamp)
	assert.Equal(t, int64(1795417681154), fd.BookTimestamp.UnixMilli())
}
