//nolint:all
package orangex

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func feedNames(n int) []string {
	feeds := make([]string, 0, n)
	for i := 0; i < n; i++ {
		feeds = append(feeds, fmt.Sprintf("T%03d-USDT", i))
	}
	return feeds
}

func TestBuildSubscriptions_ChunksAtLimit(t *testing.T) {
	for _, tc := range []struct {
		feeds     int
		wantSizes []int
	}{
		{feeds: 0, wantSizes: []int{}},
		{feeds: 1, wantSizes: []int{1}},
		{feeds: 30, wantSizes: []int{30}},
		{feeds: 31, wantSizes: []int{30, 1}},
		{feeds: 87, wantSizes: []int{30, 30, 27}},
	} {
		t.Run(fmt.Sprintf("%d feeds", tc.feeds), func(t *testing.T) {
			subs := BuildSubscriptions(feedNames(tc.feeds))

			sizes := []int{}
			seen := map[string]bool{}
			for i, sub := range subs {
				sizes = append(sizes, len(sub.Params.Channels))
				assert.LessOrEqual(t, len(sub.Params.Channels), MaxChannelsPerConnection)
				assert.Equal(t, "2.0", sub.JsonRPC)
				assert.Equal(t, "/public/subscribe", sub.Method)
				assert.Equal(t, i+1, sub.ID)
				for _, ch := range sub.Params.Channels {
					assert.False(t, seen[ch], "duplicate channel %s", ch)
					seen[ch] = true
				}
			}
			assert.Equal(t, tc.wantSizes, sizes)
			assert.Len(t, seen, tc.feeds)
		})
	}
}

func TestBuildSubscriptions_ChannelFormatAndOrder(t *testing.T) {
	subs := BuildSubscriptions([]string{"ETH-USDT", "BTC-USDT"})
	require.Len(t, subs, 1)
	assert.Equal(t, []string{"ticker.BTC-USDT.raw", "ticker.ETH-USDT.raw"}, subs[0].Params.Channels)
}

func TestNew_OneConnectionPerChunk(t *testing.T) {
	separated := map[string][]int32{}
	for i, feed := range feedNames(87) {
		separated[feed] = []int32{int32(i)}
	}

	fetcher, err := New(context.Background(), common.WithFeedMaps(common.FeedMaps{Separated: separated}))
	require.NoError(t, err)

	conns := fetcher.(*OrangeXFetcher).Conns
	require.Len(t, conns, 3)

	total := 0
	for _, ws := range conns {
		assert.Equal(t, URL, ws.Endpoint)
		require.Len(t, ws.Subscriptions, 1)
		raw, ok := ws.Subscriptions[0].([]byte)
		require.True(t, ok, "subscription must be pre-marshalled bytes")

		var sub Subscription
		require.NoError(t, json.Unmarshal(raw, &sub))
		assert.LessOrEqual(t, len(sub.Params.Channels), MaxChannelsPerConnection)
		total += len(sub.Params.Channels)
	}
	assert.Equal(t, 87, total)
}

func toMessage(t *testing.T, raw string) map[string]any {
	t.Helper()
	var message map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &message))
	return message
}

func TestParseErrorResponse_SubscriptionLimit(t *testing.T) {
	// verbatim server reply to an 87-channel subscribe
	message := toMessage(t, `{"id":"1","jsonrpc":"2.0","usIn":1791866634698,"usOut":1791866634699,"usDiff":1,"error":{"code":1005,"message":"Subscription channel limit exceeded","data":{"limit":"30","requestedNew":"87","projected":"87","current":"0"}}}`)

	response, isError := ParseErrorResponse(message)
	require.True(t, isError)
	assert.Equal(t, "1", response.ID)
	assert.Equal(t, 1005, response.Error.Code)
	assert.Equal(t, "Subscription channel limit exceeded", response.Error.Message)
	assert.Equal(t, map[string]any{"limit": "30", "requestedNew": "87", "projected": "87", "current": "0"}, response.Error.Data)
}

func TestParseErrorResponse_AutoClose(t *testing.T) {
	message := toMessage(t, `{"id":"-1","jsonrpc":"2.0","usIn":1791633251125,"usOut":1791633251125,"usDiff":0,"error":{"code":9901,"message":"websocket closing connection automatically"}}`)

	response, isError := ParseErrorResponse(message)
	require.True(t, isError)
	assert.Equal(t, ErrCodeAutoClose, response.Error.Code)
	assert.Equal(t, "websocket closing connection automatically", response.Error.Message)
	assert.Nil(t, response.Error.Data)
}

func TestParseErrorResponse_NonErrors(t *testing.T) {
	for name, raw := range map[string]string{
		"subscribe ack": `{"id":"1","jsonrpc":"2.0","result":["ticker.BTC-USDT.raw"]}`,
		"ping reply":    `{"id":"0","jsonrpc":"2.0","result":{}}`,
		"ticker":        `{"params":{"data":{"instrument_name":"BTC-USDT","last_price":"1"},"channel":"ticker.BTC-USDT.raw"},"method":"subscription","jsonrpc":"2.0"}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, isError := ParseErrorResponse(toMessage(t, raw))
			assert.False(t, isError)
		})
	}
}

func TestHandleMessage_ErrorResponseIsNotFatal(t *testing.T) {
	buffer := make(chan *common.FeedData, 1)
	f := &OrangeXFetcher{FeedMap: map[string][]int32{"BTC-USDT": {1}}, FeedDataBuffer: buffer}

	message := toMessage(t, `{"id":"2","jsonrpc":"2.0","error":{"code":1005,"message":"Subscription channel limit exceeded","data":{"limit":"30","requestedNew":"30","projected":"60","current":"30"}}}`)
	require.NoError(t, f.handleMessage(context.Background(), message))
	assert.Len(t, buffer, 0)
}

func TestHandleMessage_Ticker(t *testing.T) {
	buffer := make(chan *common.FeedData, 2)
	f := &OrangeXFetcher{FeedMap: map[string][]int32{"BTC-USDT": {1, 2}}, FeedDataBuffer: buffer}

	message := toMessage(t, `{"params":{"data":{"timestamp":"1791866634698","stats":{"volume":"12.5"},"last_price":"65000.5","instrument_name":"BTC-USDT"},"channel":"ticker.BTC-USDT.raw"},"method":"subscription","jsonrpc":"2.0"}`)
	require.NoError(t, f.handleMessage(context.Background(), message))
	require.Len(t, buffer, 2)

	got := <-buffer
	assert.Equal(t, int32(1), got.FeedID)
	assert.Equal(t, 12.5, got.Volume)
	assert.Equal(t, int64(1791866634698), got.Timestamp.UnixMilli())
}
