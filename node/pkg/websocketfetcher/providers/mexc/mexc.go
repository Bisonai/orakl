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

	params := []string{}

	for feed := range fetcher.FeedMap {
		param := "spot@public.miniTicker.v3.api@" + feed + "@UTC+0"
		params = append(params, param)
	}

	subscription := Subscription{
		Method: "SUBSCRIPTION",
		Params: params,
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
	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Mexc").Err(err).Msg("error in mexc.handleMessage")
		return err
	}

	feedData, err := ResponseToFeedData(response, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Mexc").Err(err).Msg("error in mexc.handleMessage")
		return err
	}

	f.FeedDataBuffer <- *feedData
	return nil
}

func (f *MexcFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}
