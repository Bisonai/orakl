package bithumb

import (
	"context"
	"strings"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type BithumbFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &BithumbFetcher{}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	symbols := []string{}

	for feed := range fetcher.FeedMap {
		symbol := strings.ReplaceAll(feed, "-", "_")
		symbols = append(symbols, symbol)
	}

	tickerSubscription := Subscription{
		Type:      "ticker",
		Symbols:   symbols,
		TickTypes: []string{"24H"},
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions([]any{tickerSubscription}),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Bithumb").Err(err).Msg("error in bithumb.New")
		return nil, err
	}

	fetcher.Ws = ws
	return fetcher, nil

}

func (f *BithumbFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	response, err := common.MessageToStruct[RawResponse](message)
	if err != nil {
		log.Error().Str("Player", "Bithumb").Err(err).Msg("error in bithumb.handleMessage")
		return err
	}

	if response.Type != "ticker" {
		return nil
	}

	tickerResponse, err := common.MessageToStruct[TickerResponse](message)
	if err != nil {
		log.Error().Str("Player", "Bithumb").Err(err).Msg("error in bithumb.handleMessage")
		return err
	}

	feedData, err := TickerResponseToFeedData(tickerResponse, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Bithumb").Err(err).Msg("error in bithumb.handleMessage")
		return err
	}

	f.FeedDataBuffer <- *feedData

	return nil
}

func (f *BithumbFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}
