package gateio

import (
	"context"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/utils/arr"
	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type GateioFetcher struct {
	common.Fetcher
	quotes *common.QuoteCache
}

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &GateioFetcher{quotes: common.NewQuoteCache()}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	payload := []string{}
	for feed := range fetcher.FeedMap {
		payload = append(payload, strings.ReplaceAll(feed, "-", "_"))
	}

	maxBatchSize := 10
	splittedPayloads := arr.SplitByChunkSize(payload, maxBatchSize)
	channel := "spot.tickers"
	event := "subscribe"

	subscriptions := make([]any, 0, 2*len(splittedPayloads))
	for _, ch := range []string{channel, BookTickerChannel} {
		for _, sp := range splittedPayloads {
			subscriptions = append(subscriptions, Subscription{
				Time:    time.Now().Unix(),
				Channel: ch,
				Event:   event,
				Payload: sp,
			})
		}
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions(subscriptions),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Gateio").Err(err).Msg("error in gateio.New")
		return nil, err
	}
	fetcher.Ws = ws
	return fetcher, nil
}

func (f *GateioFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	if message["channel"] == BookTickerChannel {
		return f.handleBookTicker(message)
	}
	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Gateio").Err(err).Msg("error in MessageToResponse")
		return err
	}

	if response.Channel != "spot.tickers" || response.Result.Last == "" {
		return nil
	}

	feedDataList, err := ResponseToFeedData(response, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Gateio").Err(err).Msg("error in ResponseToFeedData")
		return err
	}

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- f.quotes.ApplyTrade(feedData)
	}

	return nil
}

func (f *GateioFetcher) handleBookTicker(message map[string]any) error {
	book, err := common.MessageToStruct[BookTickerResponse](message)
	if err != nil {
		log.Error().Str("Player", "Gateio").Err(err).Msg("error in gateio.handleBookTicker")
		return err
	}
	if book.Event != "update" {
		return nil
	}
	ids, exists := f.FeedMap[strings.Replace(book.Result.CurrencyPair, "_", "-", 1)]
	if !exists {
		return nil
	}
	bid, err := common.PriceStringToFloat64(book.Result.Bid)
	if err != nil {
		return nil
	}
	ask, err := common.PriceStringToFloat64(book.Result.Ask)
	if err != nil {
		return nil
	}
	for _, feedData := range f.quotes.ApplyBook(ids, bid, ask, time.UnixMilli(book.Result.Timestamp)) {
		f.FeedDataBuffer <- feedData
	}
	return nil
}

func (f *GateioFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}
