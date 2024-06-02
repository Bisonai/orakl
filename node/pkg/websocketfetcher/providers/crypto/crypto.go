package crypto

import (
	"context"
	"strings"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type CryptoDotComFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &CryptoDotComFetcher{}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	channels := []string{}

	for feed := range fetcher.FeedMap {
		symbol := strings.ReplaceAll(feed, "-", "_")

		channels = append(channels, "ticker."+symbol)
	}

	subscription := Subscription{
		Method: "subscribe",
		Params: struct {
			Channels []string `json:"channels"`
		}{
			Channels: channels,
		},
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions([]any{subscription}),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "CryptoDotCom").Err(err).Msg("error in cryptodotcom.New")
		return nil, err
	}

	fetcher.Ws = ws
	return fetcher, nil
}

func (f *CryptoDotComFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "CryptoDotCom").Err(err).Msg("error in cryptodotcom.handleMessage")
		return err
	}

	if response.Result.Channel != "ticker" {
		return nil
	}

	feedDataList, err := ResponseToFeedDataList(response, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "CryptoDotCom").Err(err).Msg("error in cryptodotcom.handleMessage")
		return err
	}

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- *feedData
	}

	return nil
}

func (f *CryptoDotComFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}
