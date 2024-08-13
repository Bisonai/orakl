package okx

import (
	"context"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type OkxFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &OkxFetcher{}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	args := []Arg{}
	for feed := range fetcher.FeedMap {
		arg := Arg{
			Channel: "tickers",
			InstId:  feed,
		}
		args = append(args, arg)
	}

	subscription := Subscription{
		Operation: "subscribe",
		Args:      args,
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions([]any{subscription}),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Okx").Err(err).Msg("error in okx.New")
		return nil, err
	}

	fetcher.Ws = ws
	return fetcher, nil
}

func (f *OkxFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	raw, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Okx").Err(err).Msg("error in okx.handleMessage")
		return err
	}

	if len(raw.Data) == 0 {
		return nil
	}

	feedDataList := ResponseToFeedData(raw, f.FeedMap)

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- feedData
	}

	return nil
}

func (f *OkxFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}
