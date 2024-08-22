package coinex

import (
	"context"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type CoinexFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &CoinexFetcher{}
	fetcher.FeedMap = config.FeedMaps.Combined
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	params := []string{}

	for feed := range fetcher.FeedMap {
		params = append(params, feed)
	}

	subscription := Subscription{
		Method: "state.subscribe",
		Params: params,
		ID:     1,
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions([]any{subscription}),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Coinex").Err(err).Msg("error in coinex.New")
		return nil, err
	}

	fetcher.Ws = ws
	return fetcher, nil
}

func (f *CoinexFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Coinex").Err(err).Msg("error in MessageToResponse")
		return err
	}
	if response.Method != "state.update" {
		return nil
	}
	feedDataList, err := ResponseToFeedDataList(response, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Coinex").Err(err).Msg("error in ResponseToFeedDataList")
		return err
	}

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- feedData
	}
	return nil
}

func (f *CoinexFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}
