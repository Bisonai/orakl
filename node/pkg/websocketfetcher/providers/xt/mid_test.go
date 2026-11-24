package xt

import (
	"context"
	"encoding/json"
	"testing"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func handleRaw(t *testing.T, f *XtFetcher, raw string) {
	t.Helper()
	var msg map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &msg))
	require.NoError(t, f.handleMessage(context.Background(), msg))
}

func TestHandleMessage_MergesTickerAndDepth(t *testing.T) {
	buf := make(chan *common.FeedData, 10)
	f := &XtFetcher{quotes: common.NewQuoteCache()}
	f.FeedMap = map[string][]int32{"DOT-USDT": {1}}
	f.FeedDataBuffer = buf

	depth := `{"topic":"depth","event":"depth@dot_usdt,5","data":{"s":"dot_usdt","t":1795417749691,"i":1795417749691,"a":[["0.679","513.43"],["0.680","3567.82"]],"b":[["0.678","342.10"],["0.677","3746.92"]]}}`
	handleRaw(t, f, depth)
	assert.Len(t, buf, 0, "depth before ticker: no 24h volume yet")

	handleRaw(t, f, `{"topic":"ticker","event":"ticker@dot_usdt","data":{"s":"dot_usdt","t":1795487400000,"c":"0.6785","q":"1000"}}`)
	require.Len(t, buf, 1)
	fd := <-buf
	assert.Equal(t, 0.6785, fd.Value)
	assert.Equal(t, 1000.0, fd.Volume)
	assert.Equal(t, 0.678, fd.Bid)
	assert.Equal(t, 0.679, fd.Ask)
	require.NotNil(t, fd.BookTimestamp, "receive time, not the depth message's t")
	assert.Greater(t, fd.BookTimestamp.UnixMilli(), int64(1795417749691))
}
