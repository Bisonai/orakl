package binance

import (
	"context"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type BinanceFetcher struct {
	common.Fetcher
	quotes *common.QuoteCache
}

// expected to recieve feedmap with key having format "<base><quote>"
func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &BinanceFetcher{quotes: common.NewQuoteCache()}
	fetcher.FeedMap = config.FeedMaps.Combined
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	streams := []Stream{}
	for feed := range fetcher.FeedMap {
		symbol := strings.ToLower(feed)
		streams = append(streams, Stream(symbol+MiniTickerSuffix), Stream(symbol+DepthSuffix))
	}
	subscription := Subscription{"SUBSCRIBE", streams, 1}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions([]any{subscription}),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Binance").Err(err).Msg("error in binance.New")
		return nil, err
	}
	fetcher.Ws = ws

	return fetcher, nil
}

func (b *BinanceFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	combined, err := common.MessageToStruct[CombinedMessage](message)
	if err != nil || combined.Data == nil {
		return nil // subscription result etc.
	}

	if symbol, ok := strings.CutSuffix(combined.Stream, DepthSuffix); ok {
		return b.handleDepth(strings.ToUpper(symbol), combined.Data)
	}

	ticker, err := common.MessageToStruct[MiniTicker](combined.Data)
	if err != nil {
		log.Error().Str("Player", "Binance").Err(err).Msg("error in MessageToTicker")
		return err
	}

	if ticker.EventType != "24hrMiniTicker" {
		return nil
	}

	feedDataList, err := TickerToFeedData(ticker, b.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Binance").Err(err).Msg("error in MiniTickerToFeedData")
		return err
	}

	for _, feedData := range feedDataList {
		b.FeedDataBuffer <- b.quotes.ApplyTrade(feedData)
	}

	return nil
}

// handleDepth records the best bid/ask of a partial depth message. The
// message has no timestamp, so the receive time is the book time.
func (b *BinanceFetcher) handleDepth(symbol string, data map[string]any) error {
	ids, exists := b.FeedMap[symbol]
	if !exists {
		return nil
	}
	depth, err := common.MessageToStruct[PartialDepth](data)
	if err != nil {
		log.Error().Str("Player", "Binance").Err(err).Msg("error in binance.handleDepth")
		return err
	}
	bid, err := common.TopPrice(depth.Bids)
	if err != nil {
		return nil
	}
	ask, err := common.TopPrice(depth.Asks)
	if err != nil {
		return nil
	}
	for _, feedData := range b.quotes.ApplyBook(ids, bid, ask, time.Now()) {
		b.FeedDataBuffer <- feedData
	}
	return nil
}

func (b *BinanceFetcher) Run(ctx context.Context) {
	b.Ws.Run(ctx, b.handleMessage)
}
