package bitstamp

import (
	"context"
	"strings"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type BitstampFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &BitstampFetcher{}
	fetcher.FeedMap = config.FeedMaps.Combined
	fetcher.FeedDataBuffer = config.FeedDataBuffer
	fetcher.VolumeCacheMap = common.VolumeCacheMap{
		Map:   make(map[int32]common.VolumeCache),
		Mutex: sync.Mutex{},
	}

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

	feedDataList, err := TradeEventToFeedData(response, f.FeedMap, &f.VolumeCacheMap)
	if err != nil {
		log.Error().Str("Player", "Bitstamp").Err(err).Msg("error in TradeEventToFeedData")
		return err
	}

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- feedData
	}

	return nil
}

func (f *BitstampFetcher) Run(ctx context.Context) {
	go f.CacheVolumes()
	f.Ws.Run(ctx, f.handleMessage)
}

func (f *BitstampFetcher) CacheVolumes() {
	volumeTicker := time.NewTicker(common.VolumeFetchInterval * time.Millisecond)
	defer volumeTicker.Stop()

	err := FetchVolumes(f.FeedMap, &f.VolumeCacheMap)
	if err != nil {
		log.Error().Str("Player", "Bitstamp").Err(err).Msg("error in fetchVolumes")
	}

	for range volumeTicker.C {
		err := FetchVolumes(f.FeedMap, &f.VolumeCacheMap)
		if err != nil {
			log.Error().Str("Player", "Bitstamp").Err(err).Msg("error in fetchVolumes")
		}
	}
}
