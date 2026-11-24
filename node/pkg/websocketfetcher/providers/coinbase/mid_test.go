package coinbase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
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
	f := &CoinbaseFetcher{FeedMap: map[string][]int32{"BTC-USD": {1}}, FeedDataBuffer: buf}
	raw := `{"type":"ticker","sequence":140151341550,"product_id":"BTC-USD","price":"60392.93","open_24h":"61375.35","volume_24h":"16910.67939037","low_24h":"59366.13","high_24h":"64533.96","volume_30d":"311180.07438013","best_bid":"60392.93","best_bid_size":"0.00248369","best_ask":"60392.94","best_ask_size":"0.14328960","side":"sell","time":"2026-11-24T06:31:57.759893Z","trade_id":1145105337,"last_size":"0.00000002"}`
	require.NoError(t, f.handleMessage(context.Background(), toMsg(t, raw)))
	require.Len(t, buf, 1)
	fd := <-buf
	assert.Equal(t, 60392.93, fd.Value)
	assert.Equal(t, 16910.67939037, fd.Volume)
	assert.Equal(t, 60392.93, fd.Bid)
	assert.Equal(t, 60392.94, fd.Ask)
	require.NotNil(t, fd.BookTimestamp)
	want, _ := time.Parse(time.RFC3339Nano, "2026-11-24T06:31:57.759893Z")
	assert.True(t, fd.BookTimestamp.Equal(want))
}
