package bitget

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

type BitgetFetcher struct {
	common.Fetcher
	quotes *common.QuoteCache
}

// PingInterval: bitget expects a "ping" every 30 seconds
// https://www.bitget.com/api-doc/common/websocket-intro
const PingInterval = 30 * time.Second

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &BitgetFetcher{quotes: common.NewQuoteCache()}
	fetcher.FeedMap = config.FeedMaps.Combined
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	args := []Arg{}
	for feed := range fetcher.FeedMap {
		args = append(args,
			Arg{InstType: "SPOT", Channel: "ticker", InstId: feed},
			Arg{InstType: "SPOT", Channel: BooksChannel, InstId: feed})
	}
	subscription := Subscription{
		Op:   "subscribe",
		Args: args,
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithCustomReadFunc(fetcher.customReadFunc),
		wss.WithEndpoint(URL),
		wss.WithSubscriptions([]any{subscription}),
		wss.WithKeepalive(PingInterval, ping),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Bitget").Err(err).Msg("error in bitget.New")
		return nil, err
	}

	fetcher.Ws = ws
	return fetcher, nil
}

func (f *BitgetFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	if arg, ok := message["arg"].(map[string]any); ok && arg["channel"] == BooksChannel {
		return f.handleBooks(message)
	}

	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Bitget").Err(err).Msg("error in MessageToResponse")
		return err
	}

	if response.Action == nil || *response.Action != "snapshot" {
		return nil
	}

	feedDataList := ResponseToFeedDataList(response, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Bitget").Err(err).Msg("error in ResponseToFeedDataList")
		return err
	}
	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- f.quotes.ApplyTrade(feedData)
	}

	return nil
}

func (f *BitgetFetcher) handleBooks(message map[string]any) error {
	book, err := common.MessageToStruct[BooksResponse](message)
	if err != nil {
		log.Error().Str("Player", "Bitget").Err(err).Msg("error in bitget.handleBooks")
		return err
	}
	if book.Action == nil || *book.Action != "snapshot" {
		return nil
	}
	ids, exists := f.FeedMap[book.Arg.InstId]
	if !exists {
		return nil
	}
	for _, d := range book.Data {
		bid, err := common.TopPrice(d.Bids)
		if err != nil {
			continue
		}
		ask, err := common.TopPrice(d.Asks)
		if err != nil {
			continue
		}
		ms, err := strconv.ParseInt(d.Timestamp, 10, 64)
		if err != nil {
			continue
		}
		for _, feedData := range f.quotes.ApplyBook(ids, bid, ask, time.UnixMilli(ms)) {
			f.FeedDataBuffer <- feedData
		}
	}
	return nil
}

func (f *BitgetFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}

// ping is the client keepalive, sent every PingInterval on each connection.
func ping(ctx context.Context, w wss.ConnWriter) error {
	log.Debug().Str("Player", "Bitget").Msg("sending ping message to bitget server")
	return w.RawWrite(ctx, "ping")
}

func (f *BitgetFetcher) customReadFunc(ctx context.Context, conn *websocket.Conn) (map[string]interface{}, error) {
	var result map[string]interface{}
	_, data, err := conn.Read(ctx)
	if err != nil {
		log.Error().Str("Player", "Bitget").Err(err).Msg("error in Bitget.customReadFunc, failed to read from websocket")
		return nil, err
	}
	if string(data) == "pong" {
		log.Debug().Str("Player", "Bitget").Msg("received pong")
		return nil, nil
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Error().Str("Player", "Bitget").Err(err).Msg("error in Bitget.customReadFunc, failed to unmarshal data")
		return nil, err
	}

	return result, nil
}
