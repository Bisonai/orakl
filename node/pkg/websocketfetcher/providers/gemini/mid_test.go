package gemini

import (
	"context"
	"encoding/json"
	"testing"
	"time"

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

func TestHandleMessage_MergesTopOfBookAndTrades(t *testing.T) {
	buf := make(chan *common.FeedData, 10)
	f := &GeminiFetcher{quotes: common.NewQuoteCache(), books: map[string]*topOfBook{}}
	f.FeedMap = map[string][]int32{"BTCUSD": {1}}
	f.FeedDataBuffer = buf
	f.VolumeCacheMap = common.VolumeCacheMap{Map: map[int32]common.VolumeCache{}}
	f.VolumeCacheMap.Set(1, common.VolumeCache{UpdatedAt: time.Now(), Volume: 123})
	ctx := context.Background()

	initial := `{"eventId":1794982198399174,"events":[{"delta":"0.04471266","price":"60476.04","reason":"initial","remaining":"0.04471266","side":"bid","symbol":"BTCUSD","type":"change"},{"delta":"0.04471266","price":"60486.88","reason":"initial","remaining":"0.04471266","side":"ask","symbol":"BTCUSD","type":"change"}],"socket_sequence":1,"type":"update"}`
	require.NoError(t, f.handleMessage(ctx, toMsg(t, initial)))
	assert.Len(t, buf, 0, "book before first trade: no trade price/time yet")

	trade := `{"eventId":1794982199920277,"events":[{"amount":"0.03311451","makerSide":"bid","price":"60423.89","symbol":"BTCUSD","tid":2840141038355007,"type":"trade"}],"socket_sequence":4,"timestamp":1795501920,"timestampms":1795501920497,"type":"update"}`
	require.NoError(t, f.handleMessage(ctx, toMsg(t, trade)))
	require.Len(t, buf, 1)
	fd := <-buf
	assert.Equal(t, 60423.89, fd.Value)
	assert.Equal(t, 123.0, fd.Volume)
	assert.Equal(t, 60476.04, fd.Bid)
	assert.Equal(t, 60486.88, fd.Ask)

	// one-sided change after the throttle window: the other side is kept
	time.Sleep(common.BookEmitInterval)
	ask := `{"eventId":1794982199920300,"events":[{"delta":"-0.01","price":"60480.00","reason":"cancel","remaining":"0.1","side":"ask","symbol":"BTCUSD","type":"change"}],"socket_sequence":5,"timestampms":1795501921000,"type":"update"}`
	require.NoError(t, f.handleMessage(ctx, toMsg(t, ask)))
	require.Len(t, buf, 1, "book change emits; it is not a trade")
	fd = <-buf
	assert.Equal(t, 60423.89, fd.Value)
	assert.Equal(t, 60476.04, fd.Bid)
	assert.Equal(t, 60480.0, fd.Ask)
}

func TestHandleMessage_BookOnlyBeforeFirstTrade(t *testing.T) {
	buf := make(chan *common.FeedData, 10)
	fi, err := New(context.Background(),
		common.WithFeedMaps(common.FeedMaps{Combined: map[string][]int32{"BTCUSD": {1}}}),
		common.WithFeedDataBuffer(buf))
	require.NoError(t, err)
	f := fi.(*GeminiFetcher)
	ctx := context.Background()

	initial := `{"eventId":1794982198399174,"events":[{"delta":"0.04471266","price":"60476.04","reason":"initial","remaining":"0.04471266","side":"bid","symbol":"BTCUSD","type":"change"},{"delta":"0.04471266","price":"60486.88","reason":"initial","remaining":"0.04471266","side":"ask","symbol":"BTCUSD","type":"change"}],"socket_sequence":1,"type":"update"}`
	require.NoError(t, f.handleMessage(ctx, toMsg(t, initial)))
	assert.Len(t, buf, 0, "24h volume not known yet")

	f.VolumeCacheMap.Set(1, common.VolumeCache{UpdatedAt: time.Now(), Volume: 123})
	ask := `{"eventId":1794982199920300,"events":[{"delta":"-0.01","price":"60480.00","reason":"cancel","remaining":"0.1","side":"ask","symbol":"BTCUSD","type":"change"}],"socket_sequence":2,"timestampms":1795501921000,"type":"update"}`
	require.NoError(t, f.handleMessage(ctx, toMsg(t, ask)))
	require.Len(t, buf, 1)
	fd := <-buf
	assert.True(t, fd.BookOnly)
	assert.InDelta(t, 60478.02, fd.Value, 1e-9, "mid")
	assert.Equal(t, 123.0, fd.Volume)
	require.NotNil(t, fd.Timestamp)
	assert.Equal(t, fd.BookTimestamp, fd.Timestamp, "book (receive) time")

	trade := `{"eventId":1794982199920277,"events":[{"amount":"0.03311451","makerSide":"bid","price":"60423.89","symbol":"BTCUSD","tid":2840141038355007,"type":"trade"}],"socket_sequence":3,"timestamp":1795501920,"timestampms":1795501920497,"type":"update"}`
	require.NoError(t, f.handleMessage(ctx, toMsg(t, trade)))
	require.Len(t, buf, 1)
	fd = <-buf
	assert.False(t, fd.BookOnly)
	assert.Equal(t, 60423.89, fd.Value)
	assert.Equal(t, 60480.0, fd.Ask)
}
