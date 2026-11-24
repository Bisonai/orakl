package orangex

import (
	"encoding/json"
	"testing"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTickerCarriesBook(t *testing.T) {
	raw := `{"params":{"data":{"timestamp":"1795487394483","stats":{"volume":"6036374","price_change":"0.0696","low":"0.0272","turnover":"174564.7794","high":"0.0301"},"state":"open","last_price":"0.0292","instrument_name":"KAIA-USDT","best_bid_price":0.0290,"best_bid_amount":57466,"best_ask_price":0.0291,"best_ask_amount":1000},"channel":"ticker.KAIA-USDT.raw"},"method":"subscription","jsonrpc":"2.0"}`
	var msg map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &msg))
	resp, err := common.MessageToStruct[TickerResponse](msg)
	require.NoError(t, err)
	out, err := TickerResponseToFeedData(resp, map[string][]int32{"KAIA-USDT": {1}})
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, 0.0292, out[0].Value)
	assert.Equal(t, 0.029, out[0].Bid)
	assert.Equal(t, 0.0291, out[0].Ask)
	assert.Equal(t, int64(1795487394483), out[0].BookTimestamp.UnixMilli())
}
