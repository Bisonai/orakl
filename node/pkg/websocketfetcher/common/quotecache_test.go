package common

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuoteCache_BookBeforeTradeEmitsNothing(t *testing.T) {
	c := NewQuoteCache()
	out := c.ApplyBook([]int32{1}, 99, 101, time.Now())
	assert.Empty(t, out, "no 24h volume yet, nothing to weight")
}

func TestQuoteCache_MergesBothStreams(t *testing.T) {
	c := NewQuoteCache()
	bookTs := time.UnixMilli(1_700_000_000_000)
	c.ApplyBook([]int32{1, 2}, 99, 101, bookTs)

	tradeTs := time.UnixMilli(1_699_999_000_000)
	merged := c.ApplyTrade(&FeedData{FeedID: 1, Value: 100.5, Volume: 7, Timestamp: &tradeTs})
	assert.Equal(t, 100.5, merged.Value)
	assert.Equal(t, 7.0, merged.Volume)
	assert.Equal(t, 99.0, merged.Bid)
	assert.Equal(t, 101.0, merged.Ask)
	require.NotNil(t, merged.BookTimestamp)
	assert.True(t, merged.BookTimestamp.Equal(bookTs))

	bookTs2 := bookTs.Add(time.Second)
	out := c.ApplyBook([]int32{1, 2}, 100, 102, bookTs2)
	require.Len(t, out, 1, "feed 2 still has no ticker")
	assert.Equal(t, int32(1), out[0].FeedID)
	assert.Equal(t, 100.5, out[0].Value, "trade price kept")
	assert.Equal(t, 7.0, out[0].Volume)
	assert.True(t, out[0].Timestamp.Equal(tradeTs), "trade time kept")
	assert.Equal(t, 100.0, out[0].Bid)
	assert.True(t, out[0].BookTimestamp.Equal(bookTs2))

	// emitted values are copies
	out[0].Bid = 0
	again := c.ApplyBook([]int32{1}, 100, 102, bookTs2)
	assert.Equal(t, 100.0, again[0].Bid)
}

func TestSimpleOrderbook_TopOfBook(t *testing.T) {
	raw := `{"ty":"orderbook","cd":"KRW-WAVES","tms":1795395825983,"tas":1604149.8,"tbs":543437.5,
		"obu":[{"ap":214,"bp":212,"as":52691.9,"bs":4641.4},{"ap":215,"bp":211,"as":1,"bs":1}],"st":"REALTIME","lv":0}`
	var msg map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &msg))
	book, err := MessageToStruct[SimpleOrderbook](msg)
	require.NoError(t, err)

	ids, bid, ask, ts, err := book.TopOfBook(map[string][]int32{"WAVES-KRW": {7, 8}})
	require.NoError(t, err)
	assert.Equal(t, []int32{7, 8}, ids)
	assert.Equal(t, 212.0, bid)
	assert.Equal(t, 214.0, ask)
	assert.Equal(t, int64(1795395825983), ts.UnixMilli())

	_, _, _, _, err = book.TopOfBook(map[string][]int32{"BTC-KRW": {1}})
	assert.Error(t, err, "unknown market")

	empty := SimpleOrderbook{Code: "KRW-WAVES"}
	_, _, _, _, err = empty.TopOfBook(map[string][]int32{"WAVES-KRW": {7}})
	assert.Error(t, err, "empty book")
}

func TestSimpleOrderbook_TimestampUnits(t *testing.T) {
	want := time.UnixMilli(1795396941335)
	fm := map[string][]int32{"BTC-KRW": {1}}
	units := []SimpleOrderbookUnit{{AskPrice: 2, BidPrice: 1}}
	for name, tms := range map[string]int64{
		"upbit ms":    1795396941335,
		"bithumb µs":  1795396941335000,
		"nanoseconds": 1795396941335000000,
	} {
		_, _, _, ts, err := SimpleOrderbook{Code: "KRW-BTC", Timestamp: tms, Units: units}.TopOfBook(fm)
		require.NoError(t, err)
		assert.True(t, ts.Equal(want), name)
	}
}
