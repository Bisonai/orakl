package mexc

import (
	"context"
	"testing"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/mexc/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

// bookFrame builds a PushDataV3ApiWrapper frame with an aggre.bookTicker body
// using upstream field numbers (channel 1, symbol 3, sendTime 6, body 315;
// body: bidPrice 1, bidQuantity 2, askPrice 3, askQuantity 4).
func bookFrame(symbol, bid, ask string, sendTime int64) []byte {
	var body []byte
	body = protowire.AppendTag(body, 1, protowire.BytesType)
	body = protowire.AppendString(body, bid)
	body = protowire.AppendTag(body, 2, protowire.BytesType)
	body = protowire.AppendString(body, "100")
	body = protowire.AppendTag(body, 3, protowire.BytesType)
	body = protowire.AppendString(body, ask)
	body = protowire.AppendTag(body, 4, protowire.BytesType)
	body = protowire.AppendString(body, "50")

	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendString(b, BookTickerTopicPrefix+symbol)
	b = protowire.AppendTag(b, 315, protowire.BytesType)
	b = protowire.AppendBytes(b, body)
	b = protowire.AppendTag(b, 3, protowire.BytesType)
	b = protowire.AppendString(b, symbol)
	b = protowire.AppendTag(b, 6, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(sendTime))
	return b
}

func TestDecodeBookTicker(t *testing.T) {
	got, ok, err := DecodeBookTicker(bookFrame("KAIAUSDT", "0.0285", "0.0287", 1795417681154))
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, BookTicker{Symbol: "KAIAUSDT", BidPrice: "0.0285", AskPrice: "0.0287", SendTime: 1795417681154}, got)

	// a miniTickers frame has no field 315
	sym, send := "KAIAUSDT", int64(1)
	mini, err := proto.Marshal(&pb.PushDataV3ApiWrapper{Symbol: &sym, SendTime: &send,
		PublicMiniTickers: &pb.PublicMiniTickersV3Api{Items: []*pb.PublicMiniTickerV3Api{{Symbol: sym, Price: "1"}}}})
	require.NoError(t, err)
	_, ok, err = DecodeBookTicker(mini)
	require.NoError(t, err)
	assert.False(t, ok)

	_, _, err = DecodeBookTicker([]byte{0xff})
	assert.Error(t, err, "truncated frame")
}

func TestHandleMessage_MergesTickerAndBook(t *testing.T) {
	buf := make(chan *common.FeedData, 10)
	f := &MexcFetcher{quotes: common.NewQuoteCache()}
	f.FeedMap = map[string][]int32{"KAIAUSDT": {1}}
	f.FeedDataBuffer = buf
	ctx := context.Background()

	book, ok, err := DecodeBookTicker(bookFrame("KAIAUSDT", "0.0285", "0.0287", 1795417681154))
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, f.handleMessage(ctx, map[string]any{bookPayloadKey: book}))
	assert.Len(t, buf, 0, "book before ticker: no 24h volume yet")

	send := int64(1795417681000)
	wrapper := &pb.PushDataV3ApiWrapper{SendTime: &send, PublicMiniTickers: &pb.PublicMiniTickersV3Api{
		Items: []*pb.PublicMiniTickerV3Api{{Symbol: "KAIAUSDT", Price: "0.0286", Quantity: "1000"}}}}
	require.NoError(t, f.handleMessage(ctx, map[string]any{payloadKey: wrapper}))
	require.Len(t, buf, 1)
	fd := <-buf
	assert.Equal(t, 0.0286, fd.Value)
	assert.Equal(t, 1000.0, fd.Volume)
	assert.Equal(t, 0.0285, fd.Bid)
	assert.Equal(t, 0.0287, fd.Ask)
	assert.Equal(t, int64(1795417681154), fd.BookTimestamp.UnixMilli())
}
