package bitget

import (
	"context"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type BitgetFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &BitgetFetcher{}
	fetcher.FeedMap = config.FeedMaps.Combined
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	args := []Arg{}
	for feed := range fetcher.FeedMap {
		arg := Arg{
			InstType: "SP",
			Channel:  "ticker",
			InstId:   feed,
		}
		args = append(args, arg)
	}
	subscription := Subscription{
		Op:   "subscribe",
		Args: args,
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions([]any{subscription}),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Bitget").Err(err).Msg("error in bitget.New")
		return nil, err
	}

	fetcher.Ws = ws
	return fetcher, nil
}

func (f *BitgetFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Bitget").Err(err).Msg("error in MessageToResponse")
		return err
	}
	feedDataList, err := ResponseToFeedDataList(response, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Bitget").Err(err).Msg("error in ResponseToFeedDataList")
		return err
	}
	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- *feedData
	}

	return nil
}

func (f *BitgetFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}
