package btse

import (
	"context"
	"strings"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type BtseFetcher struct {
	common.Fetcher
	bookWs *wss.WebsocketHelper
	quotes *common.QuoteCache
}

const (
	IncreasedReadLimit    = 327690
	maxArgPerSubscription = 10
)

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &BtseFetcher{}
	// REST 24h volumes let the book emit before the first trade (sparse trades)
	fetcher.quotes = common.NewQuoteCacheWithVolumes(&fetcher.VolumeCacheMap)
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer
	fetcher.VolumeCacheMap = common.VolumeCacheMap{
		Map:   make(map[int32]common.VolumeCache),
		Mutex: sync.Mutex{},
	}

	subscriptions := []any{}
	args := []string{}
	for feed := range fetcher.FeedMap {
		args = append(args, "tradeHistoryApi:"+feed)
		if len(args) == maxArgPerSubscription {
			subscriptions = append(subscriptions, Subscription{
				Op:   "subscribe",
				Args: args,
			})
			args = []string{}
		}
	}

	if len(args) > 0 {
		subscriptions = append(subscriptions, Subscription{
			Op:   "subscribe",
			Args: args,
		})
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions(subscriptions),
		wss.WithProxyUrl(config.Proxy),
		wss.WithReadLimit(IncreasedReadLimit),
	)

	if err != nil {
		log.Error().Str("Player", "Btse").Err(err).Msg("error in btse.New")
		return nil, err
	}

	fetcher.Ws = ws

	bookSubscriptions := []any{}
	bookArgs := []string{}
	for feed := range fetcher.FeedMap {
		bookArgs = append(bookArgs, SnapshotL1Topic+feed+"_0")
		if len(bookArgs) == maxArgPerSubscription {
			bookSubscriptions = append(bookSubscriptions, Subscription{Op: "subscribe", Args: bookArgs})
			bookArgs = []string{}
		}
	}
	if len(bookArgs) > 0 {
		bookSubscriptions = append(bookSubscriptions, Subscription{Op: "subscribe", Args: bookArgs})
	}
	bookWs, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(OrderbookURL),
		wss.WithSubscriptions(bookSubscriptions),
		wss.WithProxyUrl(config.Proxy),
		wss.WithReadLimit(IncreasedReadLimit),
	)
	if err != nil {
		log.Error().Str("Player", "Btse").Err(err).Msg("error in btse.New, order book connection")
		return nil, err
	}
	fetcher.bookWs = bookWs

	return fetcher, nil
}

func (f *BtseFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	// snapshotL1 data is an object, trade history data an array: route on the
	// topic before decoding.
	if topic, _ := message["topic"].(string); strings.HasPrefix(topic, SnapshotL1Topic) {
		return f.handleSnapshotL1(message)
	}

	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Btse").Err(err).Msg("error in btse.handleMessage")
		return err
	}

	if !strings.HasPrefix(response.Topic, "tradeHistoryApi:") {
		return nil
	}

	feedDataList, err := ResponseToFeedDataList(response, f.FeedMap, &f.VolumeCacheMap)
	if err != nil {
		log.Error().Str("Player", "Btse").Err(err).Msg("error in btse.handleMessage")
		return err
	}

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- f.quotes.ApplyTrade(feedData)
	}
	return nil
}

func (f *BtseFetcher) handleSnapshotL1(message map[string]any) error {
	book, err := common.MessageToStruct[SnapshotL1Response](message)
	if err != nil {
		log.Error().Str("Player", "Btse").Err(err).Msg("error in btse.handleSnapshotL1")
		return err
	}
	ids, exists := f.FeedMap[book.Data.Symbol]
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
	for _, feedData := range f.quotes.ApplyBook(ids, bid, ask, time.UnixMilli(book.Data.Timestamp)) {
		f.FeedDataBuffer <- feedData
	}
	return nil
}

func (f *BtseFetcher) Run(ctx context.Context) {
	go f.CacheVolumes(ctx)
	go f.bookWs.Run(ctx, f.handleMessage)
	f.Ws.Run(ctx, f.handleMessage)
}

// volumeFetcher is overridable in tests.
var volumeFetcher = FetchVolumes

func (f *BtseFetcher) CacheVolumes(ctx context.Context) {
	volumeTicker := time.NewTicker(common.VolumeFetchInterval * time.Millisecond)
	defer volumeTicker.Stop()

	if ctx.Err() != nil {
		return
	}
	err := volumeFetcher(ctx, f.FeedMap, &f.VolumeCacheMap)
	if err != nil {
		log.Error().Str("Player", "Btse").Err(err).Msg("error in fetchVolumes")
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
				log.Error().Str("Player", "Btse").Err(err).Msg("error in fetchVolumes")
			}
		}
	}
}
