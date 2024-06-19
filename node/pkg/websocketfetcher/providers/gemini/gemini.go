package gemini

import (
	"context"
	"strings"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type GeminiFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &GeminiFetcher{}
	fetcher.FeedMap = config.FeedMaps.Combined
	fetcher.FeedDataBuffer = config.FeedDataBuffer
	fetcher.VolumeCacheMap = common.VolumeCacheMap{
		Map:   make(map[int32]common.VolumeCache),
		Mutex: sync.Mutex{},
	}

	symbols := []string{}
	for feed := range fetcher.FeedMap {
		symbols = append(symbols, strings.ToUpper(feed))
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL+strings.Join(symbols, ",")),
		wss.WithSubscriptions([]any{}),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Gemini").Err(err).Msg("error in gemini.New")
		return nil, err
	}

	fetcher.Ws = ws
	return fetcher, nil
}

func (f *GeminiFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Gemini").Err(err).Msg("error in MessageToResponse")
		return err
	}

	if response.Type != "update" || len(response.Events) == 0 {
		return nil
	}
	feedDataList, err := TradeResponseToFeedDataList(response, f.FeedMap, &f.VolumeCacheMap)
	if err != nil {
		log.Error().Str("Player", "Gemini").Err(err).Msg("error in TradeResponseToFeedDataList")
		return err
	}
	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- *feedData
	}
	return nil
}

func (f *GeminiFetcher) Run(ctx context.Context) {
	go f.CacheVolumes()
	f.Ws.Run(ctx, f.handleMessage)
}

func (f *GeminiFetcher) CacheVolumes() {
	volumeTimer := time.NewTimer(common.VolumeFetchInterval * time.Millisecond)

	FetchVolumes(f.FeedMap, &f.VolumeCacheMap)

	for {
		<-volumeTimer.C
		FetchVolumes(f.FeedMap, &f.VolumeCacheMap)
		volumeTimer.Reset(common.VolumeFetchInterval * time.Millisecond)
	}
}
