package korbit

import (
	"context"
	"strings"

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

	subscription := []Subscription{{
		Method:  "subscribe",
		Type:    "ticker",
		Symbols: symbols,
	}}

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
	// skip control messages (success/fail responses)
	if _, hasStatus := message["status"]; hasStatus {
		return nil
	}

	raw, err := common.MessageToStruct[Raw](message)
	if err != nil {
		log.Error().Str("Player", "Korbit").Err(err).Msg("error in MessageToRawResponse")
		return err
	}

	if raw.Type != "ticker" {
		return nil
	}

	feedDataList, err := RawToFeedData(raw, k.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Korbit").Err(err).Msg("error in RawToFeedData")
		return err
	}

	for _, feedData := range feedDataList {
		k.FeedDataBuffer <- feedData
	}

	return nil
}

func (k *KorbitFetcher) Run(ctx context.Context) {
	k.Ws.Run(ctx, k.handleMessage)
}
