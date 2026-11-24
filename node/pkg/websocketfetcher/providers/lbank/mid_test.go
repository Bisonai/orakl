package lbank

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

func TestHandleMessage_MergesTickAndDepth(t *testing.T) {
	buf := make(chan *common.FeedData, 10)
	f := &LbankFetcher{quotes: common.NewQuoteCache()}
	f.FeedMap = map[string][]int32{"DOT-USDT": {1}}
	f.FeedDataBuffer = buf
	ctx := context.Background()

	depth := `{"depth":{"asks":[["0.690","9810.9778"],["0.691","69594.1866"]],"bids":[["0.689","30391.9371"],["0.688","71708.0190"]]},"s":"t-04","SERVER":"V2","count":5,"type":"depth","pair":"dot_usdt","TS":"2026-11-24T14:07:10.695"}`
	require.NoError(t, f.handleMessage(ctx, toMsg(t, depth)))
	assert.Len(t, buf, 0, "depth before tick")

	tick := `{"tick":{"to_cny":6.66,"high":0.7215,"vol":4013573.2,"low":0.6677,"change":2.6,"usd":0.6895,"to_usd":1,"dir":"buy","turnover":2774510.1,"latest":0.6895,"cny":4.59},"type":"tick","pair":"dot_usdt","SERVER":"V2","TS":"2026-11-24T14:07:11.301"}`
	require.NoError(t, f.handleMessage(ctx, toMsg(t, tick)))
	require.Len(t, buf, 1)
	fd := <-buf
	assert.Equal(t, 0.6895, fd.Value)
	assert.Equal(t, 0.689, fd.Bid)
	assert.Equal(t, 0.690, fd.Ask)
	require.NotNil(t, fd.BookTimestamp)
	assert.Equal(t, "2026-11-24T06:07:10.695Z", fd.BookTimestamp.UTC().Format("2006-01-02T15:04:05.000Z"), "TS is Asia/Shanghai")
}

func TestSubscriptions_TickAndDepth(t *testing.T) {
	b, err := json.Marshal(Subscription{Action: "subscribe", Subscribe: "depth", Pair: "dot_usdt", Depth: "5"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"action":"subscribe","subscribe":"depth","pair":"dot_usdt","depth":"5"}`, string(b))
	b, err = json.Marshal(Subscription{Action: "subscribe", Subscribe: "tick", Pair: "dot_usdt"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"action":"subscribe","subscribe":"tick","pair":"dot_usdt"}`, string(b))
}
