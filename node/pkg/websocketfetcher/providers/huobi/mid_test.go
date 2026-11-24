package huobi

import (
	"encoding/json"
	"testing"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTickerCarriesBook(t *testing.T) {
	raw := `{"ch":"market.dotusdt.ticker","ts":1795487475800,"tick":{"open":0.6588,"high":0.7206,"low":0.6565,"close":0.6811,"amount":15372.15411673901,"vol":10639.40829967629,"count":400,"bid":0.6755,"bidSize":3619.9118,"ask":0.6784,"askSize":1092.8261,"lastPrice":0.6811,"lastSize":29.9911}}`
	var msg map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &msg))
	resp, err := common.MessageToStruct[Response](msg)
	require.NoError(t, err)
	out, err := ResponseToFeedData(resp, map[string][]int32{"DOTUSDT": {1}})
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, 0.6811, out[0].Value)
	assert.Equal(t, 0.6755, out[0].Bid)
	assert.Equal(t, 0.6784, out[0].Ask)
	assert.Equal(t, int64(1795487475800), out[0].BookTimestamp.UnixMilli())
}
