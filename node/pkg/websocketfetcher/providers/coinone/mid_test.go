package coinone

import (
	"context"
	"encoding/json"
	"testing"

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
	f := &CoinoneFetcher{FeedMap: map[string][]int32{"BTC-KRW": {1}}, FeedDataBuffer: buf}
	raw := `{"r":"DATA","c":"TICKER","d":{"qc":"KRW","tc":"BTC","t":1795501913967,"qv":"108330799037.5718","tv":"1286.94218003","hi":"84030000","lo":"81090000","fi":"83640000","la":"82260000","vp":"0","abp":"82270000","abq":"0.14858339","bbp":"82250000","bbq":"0.06981066","i":"1795501913967001","yhi":"87670000","ylo":"81440000","yfi":"83610000","yla":"83650000","yqv":"58256595883.3926","ytv":"701.03876331"}}`
	require.NoError(t, f.handleMessage(context.Background(), toMsg(t, raw)))
	require.Len(t, buf, 1)
	fd := <-buf
	assert.Equal(t, 82260000.0, fd.Value)
	assert.Equal(t, 1286.94218003, fd.Volume)
	assert.Equal(t, 82250000.0, fd.Bid)
	assert.Equal(t, 82270000.0, fd.Ask)
	require.NotNil(t, fd.BookTimestamp)
	assert.Equal(t, int64(1795501913967), fd.BookTimestamp.UnixMilli())
}
