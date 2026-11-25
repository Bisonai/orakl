package bingx

import (
	"context"
	"encoding/json"
	"fmt"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

type BingxFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &BingxFetcher{}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	subscriptions := []any{}
	for feed := range fetcher.FeedMap {
		subscriptions = append(subscriptions, Subscription{
			ID:          uuid.New().String(),
			RequestType: "sub",
			DataType:    fmt.Sprintf("%s@ticker", feed),
		})
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithCustomReadFunc(fetcher.customReadFunc),
		wss.WithEndpoint(URL),
		wss.WithSubscriptions(subscriptions),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Bingx").Err(err).Msg("error in bingx.New")
		return nil, err
	}
	fetcher.Ws = ws
	return fetcher, nil
}

func (f *BingxFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	if _, exists := message["ping"]; exists {
		heartbeat, err := common.MessageToStruct[Heartbeat](message)
		if err != nil {
			log.Error().Str("Player", "Bingx").Err(err).Msg("error in bingx.handleMessage, failed to parse heartbeat")
			return err
		}
		// reply on the connection the ping came on
		w, ok := wss.WriterFromContext(ctx)
		if !ok {
			return wss.ErrNoConnWriter
		}
		err = w.Write(ctx, HeartbeatResonse{
			Pong: heartbeat.Ping,
			Time: heartbeat.Time,
		})
		if err != nil {
			log.Error().Str("Player", "Bingx").Err(err).Msg("failed to resond to heartbeat, failed to write to websocket")
			return err
		}
	} else {
		if _, exists := message["data"]; !exists {
			return nil
		}
		raw, err := common.MessageToStruct[Response](message)
		if err != nil {
			log.Error().Str("Player", "Bingx").Err(err).Msg("error in bingx.handleMessage, failed to parse response")
			return err
		}
		feedDataList, err := ResponseToFeedData(raw, f.FeedMap)
		if err != nil {
			log.Error().Str("Player", "Bingx").Err(err).Msg("error in bingx.handleMessage, failed to convert response to feed data")
			return err
		}

		for _, feedData := range feedDataList {
			f.FeedDataBuffer <- feedData
		}
	}
	return nil

}

func (f *BingxFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}

func (f *BingxFetcher) customReadFunc(ctx context.Context, conn *websocket.Conn) (map[string]interface{}, error) {
	var result map[string]interface{}
	_, data, err := conn.Read(ctx)
	if err != nil {
		log.WithLevel(wss.ReadErrorLevel(ctx, zerolog.WarnLevel)).Str("Player", "Bingx").Err(err).Msg("error in bingx.customReadFunc, failed to read from websocket")
		return nil, err
	}

	decompressed, err := common.DecompressGzip(data)
	if err != nil {
		log.Error().Str("Player", "Bingx").Err(err).Msg("error in bingx.customReadFunc, failed to decompress data")
		return nil, err
	}

	err = json.Unmarshal(decompressed, &result)
	if err != nil {
		log.Error().Str("Player", "Bingx").Err(err).Msg("error in bingx.customReadFunc, failed to unmarshal data")
		return nil, err
	}

	return result, nil
}
