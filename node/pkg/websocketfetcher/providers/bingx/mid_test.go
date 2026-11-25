package bingx

import (
	"encoding/json"
	"testing"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// "A"/"B" are the best ask/bid prices, "a"/"b" their sizes: the sizes must not
// overwrite the prices (case-insensitive JSON matching).
func TestTickerCarriesBook(t *testing.T) {
	raw := `{"code":0,"timestamp":1795487678578,"dataType":"DOT-USDT@ticker","data":{"e":"24hTicker","E":1795487678577,"s":"DOT-USDT","p":0.025,"P":"3.83%","o":0.653,"h":0.722,"l":0.650,"c":0.678,"v":241201.22,"q":165621.23,"O":1795401278576,"C":1795487678577,"A":0.677,"a":32.51,"B":0.676,"b":1.47}}`
	var msg map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &msg))
	resp, err := common.MessageToStruct[Response](msg)
	require.NoError(t, err)
	out, err := ResponseToFeedData(resp, map[string][]int32{"DOT-USDT": {1}})
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, 0.678, out[0].Value)
	assert.Equal(t, 241201.22, out[0].Volume)
	assert.Equal(t, 0.676, out[0].Bid)
	assert.Equal(t, 0.677, out[0].Ask)
	assert.Equal(t, int64(1795487678577), out[0].BookTimestamp.UnixMilli())
}
