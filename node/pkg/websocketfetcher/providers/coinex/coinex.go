package coinex

import (
	"context"
	"encoding/json"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

type CoinexFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &CoinexFetcher{}
	fetcher.FeedMap = config.FeedMaps.Combined
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	params := []string{}

	for feed := range fetcher.FeedMap {
		params = append(params, feed)
	}

	subscription := Subscription{
		Method: "state.subscribe",
		Params: SubscribeParams{
			MarketList: params,
		},
		ID: 1,
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions([]any{subscription}),
		wss.WithProxyUrl(config.Proxy),
		wss.WithCustomReadFunc(fetcher.customReadFunc),
	)
	if err != nil {
		log.Error().Str("Player", "Coinex").Err(err).Msg("error in coinex.New")
		return nil, err
	}

	fetcher.Ws = ws
	return fetcher, nil
}

func (f *CoinexFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Coinex").Err(err).Msg("error in MessageToResponse")
		return err
	}
	if response.Method != "state.update" {
		return nil
	}
	feedDataList, err := ResponseToFeedDataList(response, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Coinex").Err(err).Msg("error in ResponseToFeedDataList")
		return err
	}

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- feedData
	}
	return nil
}

func (f *CoinexFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}

func (f *CoinexFetcher) customReadFunc(ctx context.Context, conn *websocket.Conn) (map[string]interface{}, error) {
	var result map[string]interface{}
	_, data, err := conn.Read(ctx)
	if err != nil {
		log.Warn().Str("Player", "coinex").Err(err).Msg("error in coinex.customReadFunc, failed to read from websocket")
		return nil, err
	}

	decompressed, err := common.DecompressGzip(data)
	if err != nil {
		log.Error().Str("Player", "coinex").Err(err).Msg("error in coinex.customReadFunc, failed to decompress data")
		return nil, err
	}

	err = json.Unmarshal(decompressed, &result)
	if err != nil {
		log.Error().Str("Player", "coinex").Err(err).Msg("error in coinex.customReadFunc, failed to unmarshal data")
		return nil, err
	}

	return result, nil
}
