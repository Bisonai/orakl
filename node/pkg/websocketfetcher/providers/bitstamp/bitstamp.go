package bitstamp

import (
	"context"
	"strings"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type BitstampFetcher common.Fetcher

var volumeCacheMap = common.VolumeCacheMap{
	Map:   make(map[int32]common.VolumeCache),
	Mutex: sync.Mutex{},
}

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &BitstampFetcher{}
	fetcher.FeedMap = config.FeedMaps.Combined
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	subscriptions := []any{}
	for feed := range fetcher.FeedMap {
		subscriptions = append(subscriptions, Subscription{
			Event: "bts:subscribe",
			Data: struct {
				Channel string `json:"channel"`
			}{
				Channel: "live_trades_" + strings.ToLower(feed),
			},
		})
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions(subscriptions),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Bitstamp").Err(err).Msg("error in bitstamp.New")
		return nil, err
	}

	fetcher.Ws = ws
	return fetcher, nil
}

func (f *BitstampFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	response, err := common.MessageToStruct[TradeEvent](message)
	if err != nil {
		log.Error().Str("Player", "Bitstamp").Err(err).Msg("error in MessageToTradeEvent")
		return err
	}

	if response.Event != "trade" {
		return nil
	}

	feedData, err := TradeEventToFeedData(response, f.FeedMap, &volumeCacheMap)
	if err != nil {
		log.Error().Str("Player", "Bitstamp").Err(err).Msg("error in TradeEventToFeedData")
		return err
	}

	f.FeedDataBuffer <- *feedData

	return nil
}

func (f *BitstampFetcher) Run(ctx context.Context) {
	go f.CacheVolumes()
	f.Ws.Run(ctx, f.handleMessage)
}

func (f *BitstampFetcher) CacheVolumes() {
	volumeTicker := time.NewTicker(common.VolumeFetchInterval * time.Millisecond)
	defer volumeTicker.Stop()

	for range volumeTicker.C {
		err := FetchVolumes(f.FeedMap, &volumeCacheMap)
		if err != nil {
			log.Error().Str("Player", "Bitstamp").Err(err).Msg("error in fetchVolumes")
		}
	}
}
