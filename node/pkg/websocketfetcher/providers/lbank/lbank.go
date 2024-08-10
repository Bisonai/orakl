package lbank

import (
	"context"
	"strings"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type LbankFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &LbankFetcher{}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	subscriptions := []any{}
	for feed := range fetcher.FeedMap {
		symbol := strings.ToLower(strings.ReplaceAll(feed, "-", "_"))

		subscriptions = append(subscriptions, Subscription{
			Action:    "subscribe",
			Subscribe: "tick",
			Pair:      symbol,
		})
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions(subscriptions),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Lbank").Err(err).Msg("error in lbank.New")
		return nil, err
	}

	fetcher.Ws = ws
	return fetcher, nil
}

func (f *LbankFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	if _, exists := message["ping"]; exists {
		ping, err := common.MessageToStruct[Ping](message)
		if err != nil {
			log.Error().Str("Player", "Lbank").Err(err).Msg("error in MessageToPing")
			return err
		}
		return f.Ws.Write(ctx, Pong{
			Action: "pong",
			Pong:   ping.Ping,
		})
	}
	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Lbank").Err(err).Msg("error in MessageToResponse")
		return err
	}
	if response.Type != "tick" {
		return nil
	}
	feedData, err := ResponseToFeedData(response, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Lbank").Err(err).Msg("error in ResponseToFeedData")
		return err
	}

	f.FeedDataBuffer <- *feedData

	return nil
}

func (f *LbankFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}
