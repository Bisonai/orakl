package lbank

import (
	"context"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type LbankFetcher struct {
	common.Fetcher
	quotes *common.QuoteCache
}

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &LbankFetcher{quotes: common.NewQuoteCache()}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	subscriptions := []any{}
	for feed := range fetcher.FeedMap {
		symbol := strings.ToLower(strings.ReplaceAll(feed, "-", "_"))

		subscriptions = append(subscriptions,
			Subscription{Action: "subscribe", Subscribe: "tick", Pair: symbol},
			Subscription{Action: "subscribe", Subscribe: "depth", Pair: symbol, Depth: "5"})
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions(subscriptions),
		// 2 subscriptions per symbol; with the default 1s pause the first
		// read would start minutes after each (re)connect
		wss.WithSubscribeDelay(SubscribeDelay),
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
		// reply on the connection the ping came on
		w, ok := wss.WriterFromContext(ctx)
		if !ok {
			return wss.ErrNoConnWriter
		}
		return w.Write(ctx, Pong{
			Action: "pong",
			Pong:   ping.Ping,
		})
	}
	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Lbank").Err(err).Msg("error in MessageToResponse")
		return err
	}
	if response.Type == "depth" {
		return f.handleDepth(message)
	}
	if response.Type != "tick" {
		return nil
	}
	feedDataList, err := ResponseToFeedData(response, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Lbank").Err(err).Msg("error in ResponseToFeedData")
		return err
	}

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- f.quotes.ApplyTrade(feedData)
	}

	return nil
}

func (f *LbankFetcher) handleDepth(message map[string]any) error {
	depth, err := common.MessageToStruct[DepthResponse](message)
	if err != nil {
		log.Error().Str("Player", "Lbank").Err(err).Msg("error in lbank.handleDepth")
		return err
	}
	ids, exists := f.FeedMap[strings.ToUpper(strings.ReplaceAll(depth.Pair, "_", "-"))]
	if !exists {
		return nil
	}
	bid, err := common.TopPrice(depth.Depth.Bids)
	if err != nil {
		return nil
	}
	ask, err := common.TopPrice(depth.Depth.Asks)
	if err != nil {
		return nil
	}
	ts, err := ParseTS(depth.TS)
	if err != nil {
		ts = time.Now()
	}
	for _, feedData := range f.quotes.ApplyBook(ids, bid, ask, ts) {
		f.FeedDataBuffer <- feedData
	}
	return nil
}

func (f *LbankFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}
