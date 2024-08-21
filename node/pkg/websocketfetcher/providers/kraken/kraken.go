package kraken

import (
	"context"
	"strings"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type KrakenFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &KrakenFetcher{}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	symbols := []string{}
	for feed := range fetcher.FeedMap {
		symbol := strings.ReplaceAll(feed, "-", "/")
		symbols = append(symbols, symbol)
	}

	params := Params{
		Channel: "ticker",
		Symbol:  symbols,
	}

	subscription := Subscription{
		Method: "subscribe",
		Params: params,
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions([]any{subscription}),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Kraken").Err(err).Msg("error in kraken.New")
		return nil, err
	}
	fetcher.Ws = ws
	return fetcher, nil
}

func (f *KrakenFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	raw, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Kraken").Err(err).Msg("error in kraken.handleMessage")
		return err
	}

	if raw.Channel != "ticker" {
		return nil
	}

	feedDataList := ResponseToFeedData(raw, f.FeedMap)

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- feedData
	}
	return nil
}

func (f *KrakenFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}
