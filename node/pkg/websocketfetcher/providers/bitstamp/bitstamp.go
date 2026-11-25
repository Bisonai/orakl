package bitstamp

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type BitstampFetcher struct {
	common.Fetcher
	quotes *common.QuoteCache
}

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &BitstampFetcher{}
	// REST 24h volumes let the book emit before the first trade (sparse trades)
	fetcher.quotes = common.NewQuoteCacheWithVolumes(&fetcher.VolumeCacheMap)
	fetcher.FeedMap = config.FeedMaps.Combined
	fetcher.FeedDataBuffer = config.FeedDataBuffer
	fetcher.VolumeCacheMap = common.VolumeCacheMap{
		Map:   make(map[int32]common.VolumeCache),
		Mutex: sync.Mutex{},
	}

	subscriptions := []any{}
	for feed := range fetcher.FeedMap {
		for _, channel := range []string{"live_trades_", OrderBookChannel} {
			subscriptions = append(subscriptions, Subscription{
				Event: "bts:subscribe",
				Data: struct {
					Channel string `json:"channel"`
				}{
					Channel: channel + strings.ToLower(feed),
				},
			})
		}
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
	if channel, _ := message["channel"].(string); strings.HasPrefix(channel, OrderBookChannel) {
		return f.handleOrderBook(message)
	}
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
		f.FeedDataBuffer <- f.quotes.ApplyTrade(feedData)
	}

	return nil
}

func (f *BitstampFetcher) handleOrderBook(message map[string]any) error {
	book, err := common.MessageToStruct[OrderBookEvent](message)
	if err != nil {
		log.Error().Str("Player", "Bitstamp").Err(err).Msg("error in bitstamp.handleOrderBook")
		return err
	}
	if book.Event != "data" {
		return nil
	}
	ids, exists := f.FeedMap[strings.ToUpper(strings.TrimPrefix(book.Channel, OrderBookChannel))]
	if !exists {
		return nil
	}
	bid, err := common.TopPrice(book.Data.Bids)
	if err != nil {
		return nil
	}
	ask, err := common.TopPrice(book.Data.Asks)
	if err != nil {
		return nil
	}
	ts := time.Now()
	if us, err := strconv.ParseInt(book.Data.Microtimestamp, 10, 64); err == nil {
		ts = time.UnixMicro(us)
	}
	for _, feedData := range f.quotes.ApplyBook(ids, bid, ask, ts) {
		f.FeedDataBuffer <- feedData
	}
	return nil
}

func (f *BitstampFetcher) Run(ctx context.Context) {
	go f.CacheVolumes(ctx)
	f.Ws.Run(ctx, f.handleMessage)
}

// volumeFetcher is overridable in tests.
var volumeFetcher = FetchVolumes

func (f *BitstampFetcher) CacheVolumes(ctx context.Context) {
	volumeTicker := time.NewTicker(common.VolumeFetchInterval * time.Millisecond)
	defer volumeTicker.Stop()

	if ctx.Err() != nil {
		return
	}
	err := volumeFetcher(ctx, f.FeedMap, &f.VolumeCacheMap)
	if err != nil {
		log.Error().Str("Player", "Bitstamp").Err(err).Msg("error in fetchVolumes")
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-volumeTicker.C:
			if ctx.Err() != nil {
				return
			}
			err := volumeFetcher(ctx, f.FeedMap, &f.VolumeCacheMap)
			if err != nil {
				log.Error().Str("Player", "Bitstamp").Err(err).Msg("error in fetchVolumes")
			}
		}
	}
}
