package gateio

import (
	"context"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type GateioFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &GateioFetcher{}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	payload := []string{}
	for feed := range fetcher.FeedMap {
		payload = append(payload, strings.Replace(feed, "-", "_", 1))
	}

	channel := "spot.tickers"

	subscription := Subscription{
		Time:    time.Now().Unix(),
		Channel: channel,
		Event:   "subscribe",
		Payload: payload,
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions([]any{subscription}),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Gateio").Err(err).Msg("error in gateio.New")
		return nil, err
	}
	fetcher.Ws = ws
	return fetcher, nil
}

func (f *GateioFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Gateio").Err(err).Msg("error in MessageToResponse")
		return err
	}

	if response.Channel != "spot.tickers" || response.Result.Last == "" {
		return nil
	}

	feedData, err := ResponseToFeedData(response, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Gateio").Err(err).Msg("error in ResponseToFeedData")
		return err
	}
	f.FeedDataBuffer <- *feedData
	return nil
}

func (f *GateioFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}
