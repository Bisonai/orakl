package korbit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/wfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
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

	pairListString := []string{}
	for feed := range fetcher.FeedMap {
		raw := strings.Split(feed, "-")
		if len(raw) < 2 {
			log.Error().Str("Player", "Coinone").Msg("invalid feed name")
			return nil, fmt.Errorf("invalid feed name")
		}
		base := raw[0]
		quote := raw[1]
		pairListString = append(pairListString, fmt.Sprintf("%s_%s", strings.ToLower(base), strings.ToLower(quote)))
	}

	subscription := Subscription{
		AccessToken: nil,
		Timestamp:   time.Now().Unix(),
		Event:       "korbit:subscribe",
		Data: Data{Channels: []string{
			"ticker:" + strings.Join(pairListString, ","),
		}},
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions([]any{subscription}),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Korbit").Err(err).Msg("error in NewWebsocketHelper")
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

	k.FeedDataBuffer <- *feedData
	return nil
}

func (k *KorbitFetcher) Run(ctx context.Context) {
	k.Ws.Run(ctx, k.handleMessage)
}
