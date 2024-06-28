package bybit

import (
	"context"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type BybitFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &BybitFetcher{}
	fetcher.FeedMap = config.FeedMaps.Combined
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	pairList := []string{}

	for feed := range fetcher.FeedMap {
		pairList = append(pairList, "tickers."+feed)
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

	if response.Topic == nil || !strings.HasPrefix(*response.Topic, "tickers.") {
		return nil
	}

	feedData, err := ResponseToFeedData(response, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Bybit").Err(err).Msg("error in bybit.handleMessage")
		return err
	}

	f.FeedDataBuffer <- *feedData
	return nil
}

func (f *BybitFetcher) Run(ctx context.Context) {
	go f.ping(ctx)
	f.Ws.Run(ctx, f.handleMessage)
}

func (f *BybitFetcher) ping(ctx context.Context) {
	// bybit expects ping message every 20seconds for stable subscription
	// https://bybit-exchange.github.io/docs/v5/ws/connect#how-to-send-the-heartbeat-packet
	ticker := time.NewTicker(20 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				log.Debug().Str("Player", "Bybit").Msg("sending ping message to bybit server")
				err := f.Ws.Write(ctx, Heartbeat{
					Op: "ping",
				})
				if err != nil {
					log.Error().Str("Player", "Bybit").Err(err).Msg("error in bybit.ping")
				}
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}
