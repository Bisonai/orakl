package mexc

import (
	"context"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type MexcFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &MexcFetcher{}
	fetcher.FeedMap = config.FeedMaps.Combined
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	subscription := Subscription{
		Method: "SUBSCRIPTION",
		Params: []string{"spot@public.miniTickers.v3.api@UTC+0"},
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions([]any{subscription}),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Mexc").Err(err).Msg("error in mexc.New")
		return nil, err
	}
	fetcher.Ws = ws
	return fetcher, nil
}

func (f *MexcFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	response, err := common.MessageToStruct[BatchResponse](message)
	if err != nil {
		log.Error().Str("Player", "Mexc").Err(err).Msg("failed to parse message to response")
		return err
	}

	feedDataList, err := ResponseToFeedDataList(response, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Mexc").Err(err).Msg("failed to extract feedData from response")
		return err
	}

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- feedData
	}

	return nil
}

func (f *MexcFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}
