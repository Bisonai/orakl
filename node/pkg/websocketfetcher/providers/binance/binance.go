package binance

import (
	"context"
	"strings"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type BinanceFetcher common.Fetcher

// expected to recieve feedmap with key having format "<base><quote>"
func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &BinanceFetcher{}
	fetcher.FeedMap = config.FeedMaps.Combined
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	streams := []Stream{}
	for feed := range fetcher.FeedMap {
		streams = append(streams, Stream(strings.ToLower(feed)+"@miniTicker"))
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
	ticker, err := common.MessageToStruct[MiniTicker](message)
	if err != nil {
		log.Error().Str("Player", "Binance").Err(err).Msg("error in MessageToTicker")
		return err
	}

	if ticker.EventType != "24hrMiniTicker" {
		return nil
	}

	feedData, err := TickerToFeedData(ticker, b.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Binance").Err(err).Msg("error in MiniTickerToFeedData")
		return err
	}

	b.FeedDataBuffer <- *feedData
	return nil
}

func (b *BinanceFetcher) Run(ctx context.Context) {
	b.Ws.Run(ctx, b.handleMessage)
}
