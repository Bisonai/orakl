package bybit

import (
	"context"
	"strings"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type BybitFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &BybitFetcher{}
	fetcher.FeedMap = config.FeedMaps.Combined
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	pairList := []string{}

	for feed := range fetcher.FeedMap {
		pairList = append(pairList, "tickers."+feed)
	}

	subscriptions := []any{}
	for i := 0; i < len(pairList); i += 3 {
		end := common.Min(i+3, len(pairList))
		subscriptions = append(subscriptions, Subscription{
			Op:   "subscribe",
			Args: pairList[i:end],
		})
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions(subscriptions),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Bybit").Err(err).Msg("error in bybit.New")
		return nil, err
	}
	fetcher.Ws = ws
	return fetcher, nil
}

func (f *BybitFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Bybit").Err(err).Msg("error in bybit.handleMessage")
		return err
	}

	if !strings.HasPrefix(response.Topic, "tickers.") || response.Data.Price == nil {
		return nil
	}

	feedData, err := ResponseToFeedData(response, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Bybit").Err(err).Msg("error in bybit.handleMessage")
		return err
	}

	f.FeedDataBuffer <- *feedData
	return nil
}

func (f *BybitFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}
