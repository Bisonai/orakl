package bybit

import (
	"context"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type BybitFetcher struct {
	common.Fetcher
	quotes *common.QuoteCache
}

// PingInterval: bybit expects a ping every 20 seconds for a stable subscription
// https://bybit-exchange.github.io/docs/v5/ws/connect#how-to-send-the-heartbeat-packet
const PingInterval = 20 * time.Second

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &BybitFetcher{quotes: common.NewQuoteCache()}
	fetcher.FeedMap = config.FeedMaps.Combined
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	pairList := []string{}

	for feed := range fetcher.FeedMap {
		pairList = append(pairList, "tickers."+feed, OrderbookTopic+feed)
	}

	subscriptions := []any{}
	// bybit allows maximum 10 pairs per subscription
	// https://bybit-exchange.github.io/docs/v5/ws/connect#public-channel---args-limits
	for i := 0; i < len(pairList); i += 10 {
		end := common.Min(i+10, len(pairList))
		subscriptions = append(subscriptions, Subscription{
			Op:   "subscribe",
			Args: pairList[i:end],
		})
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions(subscriptions),
		wss.WithKeepalive(PingInterval, ping),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Bybit").Err(err).Msg("error in bybit.New")
		return nil, err
	}
	fetcher.Ws = ws
	return fetcher, nil
}

func (f *BybitFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Bybit").Err(err).Msg("error in bybit.handleMessage")
		return err
	}

	if response.Topic == nil {
		return nil
	}
	if strings.HasPrefix(*response.Topic, OrderbookTopic) {
		return f.handleOrderbook(message)
	}
	if !strings.HasPrefix(*response.Topic, "tickers.") {
		return nil
	}

	feedDataList, err := ResponseToFeedData(response, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Bybit").Err(err).Msg("error in bybit.handleMessage")
		return err
	}

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- f.quotes.ApplyTrade(feedData)
	}

	return nil
}

func (f *BybitFetcher) handleOrderbook(message map[string]any) error {
	book, err := common.MessageToStruct[OrderbookResponse](message)
	if err != nil {
		log.Error().Str("Player", "Bybit").Err(err).Msg("error in bybit.handleOrderbook")
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
	for _, feedData := range f.quotes.ApplyBook(ids, bid, ask, time.UnixMilli(book.Timestamp)) {
		f.FeedDataBuffer <- feedData
	}
	return nil
}

func (f *BybitFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}

// ping is the client keepalive, sent every PingInterval on each connection.
func ping(ctx context.Context, w wss.ConnWriter) error {
	log.Debug().Str("Player", "Bybit").Msg("sending ping message to bybit server")
	return w.Write(ctx, Heartbeat{
		Op: "ping",
	})
}
