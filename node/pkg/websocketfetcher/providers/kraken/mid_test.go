package kraken

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

func TestHandleMessage_TickerBook(t *testing.T) {
	buf := make(chan *common.FeedData, 10)
	f := &KrakenFetcher{FeedMap: map[string][]int32{"DOT-USDT": {1}}, FeedDataBuffer: buf}
	raw := `{"channel":"ticker","type":"update","data":[{"symbol":"DOT/USDT","bid":0.6894,"bid_qty":368.40585143,"ask":0.6899,"ask_qty":703.37272327,"last":0.6890,"volume":110494.48123947,"vwap":0.6932,"low":0.6677,"high":0.7209,"change":0.0179,"change_pct":2.67,"trades":270,"timestamp":"2026-11-24T06:08:32.963239Z"}]}`
	before := time.Now()
	require.NoError(t, f.handleMessage(context.Background(), toMsg(t, raw)))
	require.Len(t, buf, 1)
	fd := <-buf
	assert.Equal(t, 0.6890, fd.Value)
	assert.Equal(t, 110494.48123947, fd.Volume)
	assert.Equal(t, 0.6894, fd.Bid)
	assert.Equal(t, 0.6899, fd.Ask)
	require.NotNil(t, fd.BookTimestamp)
	assert.False(t, fd.BookTimestamp.Before(before), "receive time, as the ticker's own timestamp")
}

func TestSubscriptionUsesBBOTrigger(t *testing.T) {
	b, err := json.Marshal(Params{Channel: "ticker", Symbol: []string{"DOT/USDT"}, EventTrigger: "bbo"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"channel":"ticker","symbol":["DOT/USDT"],"event_trigger":"bbo"}`, string(b))
}
