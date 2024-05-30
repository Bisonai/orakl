package coinone

import (
	"context"
	"fmt"
	"strings"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type CoinoneFetcher common.Fetcher

// expected to recieve feedmap with key having format "<base>-<quote>"
func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &CoinoneFetcher{}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	subscriptions := []any{}
	for feed := range fetcher.FeedMap {
		raw := strings.Split(feed, "-")
		if len(raw) < 2 {
			log.Error().Str("Player", "Coinone").Msg("invalid feed name")
			return nil, fmt.Errorf("invalid feed name")
		}
		base := raw[0]
		quote := raw[1]

		subscriptions = append(subscriptions, Subscription{
			RequestType: "SUBSCRIBE",
			Channel:     "TICKER",
			Topic: Topic{
				QuoteCurrency:  quote,
				TargetCurrency: base,
			},
			Format: "SHORT",
		})
	}
	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions(subscriptions),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Coinone").Err(err).Msg("error in NewWebsocketHelper")
		return nil, err
	}
	fetcher.Ws = ws

	return fetcher, nil
}

func (c *CoinoneFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	raw, err := common.MessageToStruct[Raw](message)
	if err != nil {
		log.Error().Str("Player", "Coinone").Err(err).Msg("error in MessageToRawResponse")
		return err
	}

	if raw.ResponseType != "DATA" {
		return nil
	}
	feedData, err := DataToFeedData(raw.Data, c.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Coinone").Err(err).Msg("error in DataToFeedData")
		return err
	}

	c.FeedDataBuffer <- *feedData
	return nil
}

func (c *CoinoneFetcher) Run(ctx context.Context) {
	c.Ws.Run(ctx, c.handleMessage)
}
