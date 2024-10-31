package xt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

type XtFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &XtFetcher{}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	subscriptions := []any{}
	for feed := range fetcher.FeedMap {
		symbol := strings.ReplaceAll(feed, "-", "_")
		symbol = strings.ToLower(symbol)
		params := []string{
			fmt.Sprintf("ticker@%s", symbol)}

		subscriptions = append(subscriptions, Subscription{
			Method: "subscribe",
			Params: params,
			ID:     uuid.New().String(),
		})
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithCustomReadFunc(fetcher.customReadFunc),
		wss.WithCompressionMode(),
		wss.WithEndpoint(URL),
		wss.WithSubscriptions(subscriptions),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Xt").Err(err).Msg("error in xt.New")
		return nil, err
	}

	fetcher.Ws = ws
	return fetcher, nil
}

func (f *XtFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	if message == nil {
		return nil
	}
	raw, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Xt").Err(err).Msg("error in xt.handleMessage, failed to parse response")
		return err
	}
	if raw.Topic != "ticker" {
		return nil
	}

	feedDataList, err := ResponseToFeedData(raw, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Xt").Err(err).Msg("error in xt.handleMessage, failed to convert response to feed data")
		return err
	}

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- feedData
	}

	return nil
}

func (f *XtFetcher) Run(ctx context.Context) {
	go f.ping(ctx)
	f.Ws.Run(ctx, f.handleMessage)
}

func (f *XtFetcher) ping(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		err := f.Ws.RawWrite(ctx, "ping")
		if err != nil {
			log.Error().Str("Player", "Xt").Err(err).Msg("error in xt.ping")
		}
	}
}

func (f *XtFetcher) customReadFunc(ctx context.Context, conn *websocket.Conn) (map[string]interface{}, error) {
	var result map[string]interface{}
	_, data, err := conn.Read(ctx)
	if err != nil {
		log.Error().Str("Player", "Xt").Err(err).Msg("error in xt.customReadFunc, failed to read from websocket")
		return nil, err
	}
	if string(data) == "pong" {
		log.Debug().Str("Player", "Xt").Msg("received pong")
		return nil, nil
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Error().Str("Player", "Xt").Err(err).Msg("error in xt.customReadFunc, failed to unmarshal data")
		return nil, err
	}

	return result, nil
}
