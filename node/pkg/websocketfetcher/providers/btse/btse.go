package btse

import (
	"context"
	"strings"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type BtseFetcher common.Fetcher

var volumeCacheMap = common.VolumeCacheMap{
	Map:   make(map[int32]common.VolumeCache),
	Mutex: sync.Mutex{},
}

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &BtseFetcher{}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	args := []string{}

	for feed := range fetcher.FeedMap {
		args = append(args, "tradeHistoryApi:"+feed)
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
		log.Error().Str("Player", "Btse").Err(err).Msg("error in btse.New")
		return nil, err
	}

	fetcher.Ws = ws
	return fetcher, nil
}

func (f *BtseFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Btse").Err(err).Msg("error in btse.handleMessage")
		return err
	}

	if !strings.HasPrefix(response.Topic, "tradeHistoryApi:") {
		return nil
	}

	feedDataList, err := ResponseToFeedDataList(response, f.FeedMap, &volumeCacheMap)
	if err != nil {
		log.Error().Str("Player", "Btse").Err(err).Msg("error in btse.handleMessage")
		return err
	}

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- *feedData
	}
	return nil
}

func (f *BtseFetcher) Run(ctx context.Context) {
	go f.CacheVolumes()
	f.Ws.Run(ctx, f.handleMessage)
}

func (f *BtseFetcher) CacheVolumes() {
	volumeTicker := time.NewTicker(common.VolumeFetchInterval * time.Millisecond)
	defer volumeTicker.Stop()

	for range volumeTicker.C {
		err := FetchVolumes(f.FeedMap, &volumeCacheMap)
		if err != nil {
			log.Error().Str("Player", "Btse").Err(err).Msg("error in fetchVolumes")
		}
	}
}
