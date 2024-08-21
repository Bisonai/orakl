package korbit

import (
	"context"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type KorbitFetcher common.Fetcher

// expected to recieve feedmap with key having format "<base>-<quote>"
func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &KorbitFetcher{}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	symbols := []string{}
	for feed := range fetcher.FeedMap {
		symbol := strings.ToLower(strings.ReplaceAll(feed, "-", "_"))
		symbols = append(symbols, symbol)
	}

	subscription := Subscription{
		AccessToken: nil,
		Timestamp:   time.Now().Unix(),
		Event:       "korbit:subscribe",
		Data: Data{Channels: []string{
			"ticker:" + strings.Join(symbols, ","),
		}},
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions([]any{subscription}),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Korbit").Err(err).Msg("error in korbit.New")
		return nil, err
	}
	fetcher.Ws = ws
	return fetcher, nil

}

func (k *KorbitFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	raw, err := common.MessageToStruct[Raw](message)
	if err != nil {
		log.Error().Str("Player", "Korbit").Err(err).Msg("error in MessageToRawResponse")
		return err
	}

	if raw.Event != "korbit:push-ticker" {
		return nil
	}

	feedData, err := DataToFeedData(raw.Data, k.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Korbit").Err(err).Msg("error in DataToFeedData")
		return err
	}

	k.FeedDataBuffer <- feedData
	return nil
}

func (k *KorbitFetcher) Run(ctx context.Context) {
	k.Ws.Run(ctx, k.handleMessage)
}
