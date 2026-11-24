package crypto

import (
	"encoding/json"
	"testing"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTickerCarriesBook(t *testing.T) {
	raw := `{"id":-1,"method":"subscribe","code":0,"result":{"instrument_name":"DOT_USDT","subscription":"ticker.DOT_USDT","channel":"ticker","data":[{"h":"0.7231","l":"0.6499","a":"0.6780","c":"0.0392","b":"0.6775","bs":"723.33","k":"0.6779","ks":"621.33","i":"DOT_USDT","v":"1119704.56","vv":"766800.73","oi":"0","t":1795487438018}]}}`
	var msg map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &msg))
	resp, err := common.MessageToStruct[Response](msg)
	require.NoError(t, err)
	out, err := ResponseToFeedDataList(resp, map[string][]int32{"DOT-USDT": {1}})
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, 0.678, out[0].Value)
	assert.Equal(t, 0.6775, out[0].Bid)
	assert.Equal(t, 0.6779, out[0].Ask)
	assert.Equal(t, int64(1795487438018), out[0].BookTimestamp.UnixMilli())

	// null bid/ask (e.g. empty book): no book data
	raw = `{"result":{"channel":"ticker","data":[{"a":"0.048","b":null,"k":null,"i":"KAIA_USDT","v":"0","t":1795487394615}]}}`
	require.NoError(t, json.Unmarshal([]byte(raw), &msg))
	resp, _ = common.MessageToStruct[Response](msg)
	out, _ = ResponseToFeedDataList(resp, map[string][]int32{"KAIA-USDT": {2}})
	require.Len(t, out, 1)
	assert.Nil(t, out[0].BookTimestamp)
}
