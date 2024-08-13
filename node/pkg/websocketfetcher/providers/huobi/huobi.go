package huobi

import (
	"context"
	"encoding/json"
	"strings"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

type HuobiFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &HuobiFetcher{}
	fetcher.FeedMap = config.FeedMaps.Combined
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	subscriptions := []any{}
	for feed := range fetcher.FeedMap {
		subscriptions = append(subscriptions, Subscription{
			Sub: "market." + strings.ToLower(feed) + ".ticker",
		})
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithCustomReadFunc(fetcher.customReadFunc),
		wss.WithEndpoint(URL),
		wss.WithSubscriptions(subscriptions),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Huobi").Err(err).Msg("error in huobi.New")
		return nil, err
	}
	fetcher.Ws = ws
	return fetcher, nil
}

func (f *HuobiFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	if _, exists := message["ping"]; exists {
		heartbeat, err := common.MessageToStruct[Heartbeat](message)
		if err != nil {
			log.Error().Str("Player", "Huobi").Err(err).Msg("error in huobi.handleMessage, failed to parse heartbeat")
			return err
		}
		err = f.Ws.Write(ctx, HeartbeatResponse{
			Pong: heartbeat.Ping,
		})
		if err != nil {
			log.Error().Str("Player", "Huobi").Err(err).Msg("failed to resond to heartbeat, failed to write to websocket")
			return err
		}
	} else {
		if _, exists := message["subbed"]; exists {
			_ = f.checkSubResponse(message)
			return nil
		}

		if _, exists := message["tick"]; !exists {
			return nil
		}

		response, err := common.MessageToStruct[Response](message)
		if err != nil {
			log.Error().Str("Player", "Huobi").Err(err).Msg("error in huobi.handleMessage, failed to parse response")
			return err
		}
		feedData, err := ResponseToFeedData(response, f.FeedMap)
		if err != nil {
			log.Error().Str("Player", "Huobi").Err(err).Msg("error in huobi.handleMessage, failed to convert response to feed data")
			return err
		}

		f.FeedDataBuffer <- feedData
	}

	return nil
}

func (f *HuobiFetcher) checkSubResponse(message map[string]any) error {
	subResponse, err := common.MessageToStruct[SubResponse](message)
	if err != nil {
		log.Error().Str("Player", "Huobi").Err(err).Msg("error in huobi.checkSubResponse, failed to parse sub response")
		return err
	}
	log.Debug().Str("Player", "Huobi").Any("SubResponse", subResponse).Msg("sub response received")
	if subResponse.Status != "ok" {
		log.Error().Str("Player", "Huobi").Str("Status", subResponse.Status).Msg("error in huobi.checkSubResponse, sub response status is not ok")
	}

	return nil
}

func (f *HuobiFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}

func (f *HuobiFetcher) customReadFunc(ctx context.Context, conn *websocket.Conn) (map[string]interface{}, error) {
	var result map[string]interface{}
	_, data, err := conn.Read(ctx)
	if err != nil {
		log.Error().Str("Player", "Huobi").Err(err).Msg("error in huobi.customReadFunc, failed to read from websocket")
		return nil, err
	}

	decompressed, err := common.DecompressGzip(data)
	if err != nil {
		log.Error().Str("Player", "Huobi").Err(err).Msg("error in huobi.customReadFunc, failed to decompress data")
		return nil, err
	}

	err = json.Unmarshal(decompressed, &result)
	if err != nil {
		log.Error().Str("Player", "Huobi").Err(err).Msg("error in huobi.customReadFunc, failed to unmarshal data")
		return nil, err
	}

	return result, nil
}
