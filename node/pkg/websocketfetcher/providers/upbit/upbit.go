package upbit

import (
	"context"
	"strings"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// UpbitFetcher reads tickers (last trade, 24h volume) and orderbooks (best
// bid/ask) in upbit's SIMPLE websocket format. bithumb serves the same format
// (see NewCompatible).
type UpbitFetcher struct {
	common.Fetcher
	player string
	quotes *common.QuoteCache
}

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	return NewCompatible(ctx, URL, "Upbit", opts...)
}

// NewCompatible builds a fetcher for an upbit-compatible websocket endpoint.
// player names the exchange in logs.
func NewCompatible(ctx context.Context, endpoint string, player string, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &UpbitFetcher{player: player, quotes: common.NewQuoteCache()}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	codes := []string{}
	for feed := range fetcher.FeedMap {
		splitted := strings.Split(feed, "-")
		base := strings.ToUpper(splitted[0])
		quote := strings.ToUpper(splitted[1])
		codes = append(codes, quote+"-"+base)
	}

	// Snapshot + realtime for both types: a quiet market then has its 24h
	// volume and book right after (re)connect instead of at its next trade.
	// Full-depth book (not ".1"): deeper levels change more often, so the
	// book timestamp also shows the market is live while its top is quiet.
	subscription := Subscription{
		map[string]string{"ticket": uuid.New().String()},
		map[string]interface{}{"type": "ticker", "codes": codes},
		map[string]interface{}{"type": common.SimpleOrderbookType, "codes": codes},
		map[string]string{"format": "SIMPLE"},
	}
	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(endpoint),
		wss.WithSubscriptions([]interface{}{subscription}),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", player).Err(err).Msg("error in upbit.NewCompatible")
		return nil, err
	}

	fetcher.Ws = ws
	return fetcher, nil
}

func (f *UpbitFetcher) handleMessage(ctx context.Context, message map[string]interface{}) error {
	if message["ty"] == common.SimpleOrderbookType {
		book, err := common.MessageToStruct[common.SimpleOrderbook](message)
		if err != nil {
			log.Error().Str("Player", f.player).Err(err).Msg("error in upbit.handleMessage, orderbook")
			return err
		}
		ids, bid, ask, ts, err := book.TopOfBook(f.FeedMap)
		if err != nil {
			return err
		}
		for _, feedData := range f.quotes.ApplyBook(ids, bid, ask, ts) {
			f.FeedDataBuffer <- feedData
		}
		return nil
	}

	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", f.player).Err(err).Msg("error in upbit.handleMessage")
		return err
	}
	feedDataList, err := ResponseToFeedData(response, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", f.player).Err(err).Msg("error in upbit.handleMessage")
		return err
	}

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- f.quotes.ApplyTrade(feedData)
	}

	return nil
}

func (f *UpbitFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}
