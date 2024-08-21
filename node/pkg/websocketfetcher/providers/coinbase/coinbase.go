package coinbase

import (
	"context"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type CoinbaseFetcher common.Fetcher

// expected to recieve feedmap with key having format "<base>-<quote>"
func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &CoinbaseFetcher{}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	pairListString := []string{}
	for feed := range fetcher.FeedMap {
		pairListString = append(pairListString, feed)
	}
	subscription := Subscription{
		Type:       "subscribe",
		ProductIds: pairListString,
		Channels:   []string{"ticker"},
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions([]any{subscription}),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Coinbase").Err(err).Msg("error in coinbase.New")
		return nil, err
	}
	fetcher.Ws = ws
	return fetcher, nil
}

func (c *CoinbaseFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	ticker, err := common.MessageToStruct[Ticker](message)
	if err != nil {
		return err
	}

	if ticker.Type != "ticker" {
		return nil
	}

	feedData, err := TickerToFeedData(ticker, c.FeedMap)
	if err != nil {
		return err
	}

	c.FeedDataBuffer <- feedData

	return nil
}

func (k *CoinbaseFetcher) Run(ctx context.Context) {
	k.Ws.Run(ctx, k.handleMessage)
}
