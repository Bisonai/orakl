package korbit

import (
	"context"
	"encoding/json"
	"testing"

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

func TestHandleMessage_TickerBook(t *testing.T) {
	buf := make(chan *common.FeedData, 10)
	f := &KorbitFetcher{FeedMap: map[string][]int32{"DOT-KRW": {1}}, FeedDataBuffer: buf}
	raw := `{"symbol":"dot_krw","timestamp":1795500445669,"type":"ticker","snapshot":true,"data":{"open":"926","high":"931.6","low":"926","close":"931.6","prevClose":"971.8","priceChange":"-40.2","priceChangePercent":"-4.14","volume":"37.42287446","quoteVolume":"34751.142858092","bestAskPrice":"976.1","bestBidPrice":"931.2","lastTradedAt":1795497242316}}`
	require.NoError(t, f.handleMessage(context.Background(), toMsg(t, raw)))
	require.Len(t, buf, 1)
	fd := <-buf
	assert.Equal(t, 931.6, fd.Value)
	assert.Equal(t, 931.2, fd.Bid)
	assert.Equal(t, 976.1, fd.Ask)
	require.NotNil(t, fd.BookTimestamp)
	assert.Equal(t, int64(1795500445669), fd.BookTimestamp.UnixMilli())
}
