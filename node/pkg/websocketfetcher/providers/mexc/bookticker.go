package mexc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protowire"
	"nhooyr.io/websocket"
)

const (
	// BookTickerTopicPrefix + symbol is the aggregated best bid/ask topic
	// (pushed at most every 100 ms, only on change).
	BookTickerTopicPrefix = "spot@public.aggre.bookTicker.v3.api.pb@100ms@"

	// MaxTopicsPerConnection: MEXC rejects topics past 30 on one connection
	// ("Exceeded maximum subscription limit!"), so book topics are sharded.
	MaxTopicsPerConnection = 30

	// PushDataV3ApiWrapper / PublicAggreBookTickerV3Api field numbers
	// (https://github.com/mexcdevelop/websocket-proto). Decoded with protowire
	// instead of vendoring and generating the full wrapper (see
	// proto/PushDataV3ApiWrapper.proto for why it is trimmed).
	wrapperSymbolField          = 3
	wrapperSendTimeField        = 6
	wrapperAggreBookTickerField = 315
	bookTickerBidPriceField     = 1
	bookTickerAskPriceField     = 3

	bookPayloadKey = "bookTicker"
)

// BookTicker is the decoded best bid/ask of one aggre.bookTicker frame.
type BookTicker struct {
	Symbol   string
	BidPrice string
	AskPrice string
	SendTime int64 // ms
}

// DecodeBookTicker reads symbol, sendTime and the aggre.bookTicker body of a
// PushDataV3ApiWrapper frame. ok is false for frames of other topics.
func DecodeBookTicker(data []byte) (BookTicker, bool, error) {
	var t BookTicker
	var body []byte
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return t, false, protowire.ParseError(n)
		}
		data = data[n:]
		switch {
		case num == wrapperSymbolField && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return t, false, protowire.ParseError(n)
			}
			t.Symbol, data = v, data[n:]
		case num == wrapperSendTimeField && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return t, false, protowire.ParseError(n)
			}
			t.SendTime, data = int64(v), data[n:]
		case num == wrapperAggreBookTickerField && typ == protowire.BytesType:
			v, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return t, false, protowire.ParseError(n)
			}
			body, data = v, data[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return t, false, protowire.ParseError(n)
			}
			data = data[n:]
		}
	}
	if body == nil {
		return t, false, nil
	}
	for len(body) > 0 {
		num, typ, n := protowire.ConsumeTag(body)
		if n < 0 {
			return t, false, protowire.ParseError(n)
		}
		body = body[n:]
		if typ == protowire.BytesType && (num == bookTickerBidPriceField || num == bookTickerAskPriceField) {
			v, n := protowire.ConsumeString(body)
			if n < 0 {
				return t, false, protowire.ParseError(n)
			}
			if num == bookTickerBidPriceField {
				t.BidPrice = v
			} else {
				t.AskPrice = v
			}
			body = body[n:]
			continue
		}
		n = protowire.ConsumeFieldValue(num, typ, body)
		if n < 0 {
			return t, false, protowire.ParseError(n)
		}
		body = body[n:]
	}
	if t.Symbol == "" || t.BidPrice == "" || t.AskPrice == "" {
		return t, false, fmt.Errorf("incomplete bookTicker frame (symbol %q)", t.Symbol)
	}
	return t, true, nil
}

// readBookFrame is the read func of the book ticker connections. Like
// readFrame, a bad frame is logged and skipped, not returned as an error (that
// would reconnect).
func readBookFrame(ctx context.Context, conn *websocket.Conn) (map[string]any, error) {
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		return nil, err
	}

	if messageType != websocket.MessageBinary {
		var ack Ack
		if err := json.Unmarshal(data, &ack); err != nil {
			log.Warn().Str("Player", "Mexc").Str("frame", string(data)).Msg("unrecognized text frame")
			return nil, nil
		}
		if ack.Code != 0 || strings.Contains(ack.Msg, "Not Subscribed") {
			log.Error().Str("Player", "Mexc").Int("code", ack.Code).Str("msg", ack.Msg).Msg("mexc rejected book ticker topics")
		}
		return nil, nil
	}

	ticker, ok, err := DecodeBookTicker(data)
	if err != nil {
		log.Error().Str("Player", "Mexc").Err(err).Msg("failed to decode bookTicker frame")
		return nil, nil
	}
	if !ok {
		return nil, nil
	}
	return map[string]any{bookPayloadKey: ticker}, nil
}
