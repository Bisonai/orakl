package okx

import (
	"context"
	"strconv"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type OkxFetcher struct {
	common.Fetcher
	quotes *common.QuoteCache
}

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &OkxFetcher{quotes: common.NewQuoteCache()}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	args := []Arg{}
	for feed := range fetcher.FeedMap {
		args = append(args, Arg{Channel: "tickers", InstId: feed}, Arg{Channel: BboChannel, InstId: feed})
	}

	subscription := Subscription{
		Operation: "subscribe",
		Args:      args,
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions([]any{subscription}),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Okx").Err(err).Msg("error in okx.New")
		return nil, err
	}

	fetcher.Ws = ws
	return fetcher, nil
}

func (f *OkxFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	if arg, ok := message["arg"].(map[string]any); ok && arg["channel"] == BboChannel {
		return f.handleBbo(message)
	}

	raw, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Okx").Err(err).Msg("error in okx.handleMessage")
		return err
	}

	if len(raw.Data) == 0 {
		return nil
	}

	feedDataList := ResponseToFeedData(raw, f.FeedMap)

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- f.quotes.ApplyTrade(feedData)
	}

	return nil
}

func (f *OkxFetcher) handleBbo(message map[string]any) error {
	raw, err := common.MessageToStruct[BboResponse](message)
	if err != nil {
		log.Error().Str("Player", "Okx").Err(err).Msg("error in okx.handleBbo")
		return err
	}
	ids, exists := f.FeedMap[raw.Arg.InstId]
	if !exists {
		return nil
	}
	for _, d := range raw.Data {
		bid, err := common.TopPrice(d.Bids)
		if err != nil {
			continue
		}
		ask, err := common.TopPrice(d.Asks)
		if err != nil {
			continue
		}
		ms, err := strconv.ParseInt(d.Timestamp, 10, 64)
		if err != nil {
			continue
		}
		for _, feedData := range f.quotes.ApplyBook(ids, bid, ask, time.UnixMilli(ms)) {
			f.FeedDataBuffer <- feedData
		}
	}
	return nil
}

func (f *OkxFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}
