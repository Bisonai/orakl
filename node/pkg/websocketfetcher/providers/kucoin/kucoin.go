package kucoin

import (
	"context"
	"fmt"
	"strings"

	"bisonai.com/orakl/node/pkg/utils/request"
	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type KucoinFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &KucoinFetcher{}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	pairListString := []string{}
	for feed := range fetcher.FeedMap {
		raw := strings.Split(feed, "-")
		if len(raw) < 2 {
			log.Error().Str("Player", "Kucoin").Msg("invalid feed name")
			return nil, fmt.Errorf("invalid feed name")
		}
		base := raw[0]
		quote := raw[1]
		pairListString = append(pairListString, fmt.Sprintf("%s-%s", strings.ToUpper(base), strings.ToUpper(quote)))
	}

	subscription := Subscription{
		ID:       1,
		Type:     "subscribe",
		Topic:    "/market/ticker:" + strings.Join(pairListString, ","),
		Response: true,
	}

	fmt.Println(subscription)

	resp, err := request.UrlRequest[TokenResponse](TokenUrl, "POST", nil, nil, "")
	if err != nil {
		log.Error().Str("Player", "Kucoin").Err(err).Msg("error in kucoin.New")
		return nil, err
	}
	token := resp.Data.Token
	url := URL + "?token=" + token

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(url),
		wss.WithSubscriptions([]any{subscription}),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Kucoin").Err(err).Msg("error in kucoin.New")
		return nil, err
	}
	fetcher.Ws = ws
	return fetcher, nil
}

func (f *KucoinFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	raw, err := common.MessageToStruct[Raw](message)
	if err != nil {
		log.Error().Str("Player", "Kucoin").Err(err).Msg("error in kucoin.handleMessage")
		return err
	}

	if raw.Subject != "trade.ticker" {
		return nil
	}

	feedData, err := RawDataToFeedData(raw, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Kucoin").Err(err).Msg("error in kucoin.handleMessage")
		return err
	}

	f.FeedDataBuffer <- *feedData
	return nil
}

func (f *KucoinFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}
