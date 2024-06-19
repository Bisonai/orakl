package bitmart

import (
	"context"
	"fmt"
	"strings"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type BitmartFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &BitmartFetcher{}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	args := []string{}
	for feed := range fetcher.FeedMap {
		symbol := strings.ReplaceAll(feed, "-", "_")
		args = append(args, fmt.Sprintf("spot/ticker:%s", symbol))
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
		log.Error().Str("Player", "Bitmart").Err(err).Msg("error in bitmart.New")
		return nil, err
	}

	fetcher.Ws = ws
	return fetcher, nil
}

func (f *BitmartFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Bitmart").Err(err).Msg("error in bitmart.handleMessage")
		return err
	}
	if response.Table != "spot/ticker" {
		return nil
	}

	feedDataList := ResponseToFeedData(response, f.FeedMap)

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- *feedData
	}

	return nil
}

func (f *BitmartFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}
