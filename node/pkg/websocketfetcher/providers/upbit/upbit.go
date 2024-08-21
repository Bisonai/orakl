package upbit

import (
	"context"
	"strings"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type UpbitFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &UpbitFetcher{}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	codes := []string{}
	for feed := range fetcher.FeedMap {
		splitted := strings.Split(feed, "-")
		base := strings.ToUpper(splitted[0])
		quote := strings.ToUpper(splitted[1])
		codes = append(codes, quote+"-"+base)
	}

	subscription := Subscription{
		map[string]string{"ticket": uuid.New().String()},
		// map[string]interface{}{"type": "trade", "codes": codes, "isOnlyRealtime": true},
		map[string]interface{}{"type": "ticker", "codes": codes, "isOnlyRealtime": true},
		map[string]string{"format": "SIMPLE"},
	}
	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions([]interface{}{subscription}),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Upbit").Err(err).Msg("error in upbit.New")
		return nil, err
	}

	fetcher.Ws = ws
	return fetcher, nil
}

func (f *UpbitFetcher) handleMessage(ctx context.Context, message map[string]interface{}) error {
	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Upbit").Err(err).Msg("error in upbit.handleMessage")
		return err
	}
	feedData, err := ResponseToFeedData(response, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Upbit").Err(err).Msg("error in upbit.handleMessage")
		return err
	}
	f.FeedDataBuffer <- feedData
	return nil
}

func (f *UpbitFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}
