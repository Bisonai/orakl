package bitmart

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

type BitmartFetcher common.Fetcher

const PingInterval = 10 * time.Second

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &BitmartFetcher{}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	args := []string{}
	for feed := range fetcher.FeedMap {
		symbol := strings.ReplaceAll(feed, "-", "_")
		args = append(args, fmt.Sprintf("spot/ticker:%s", symbol))
	}

	subscriptions := []any{}
	for i := 0; i < len(args); i += 3 {
		end := common.Min(i+3, len(args))
		subscriptions = append(subscriptions, Subscription{
			Operation: "subscribe",
			Args:      args[i:end],
		})
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithCustomReadFunc(fetcher.customReadFunc),
		wss.WithEndpoint(URL),
		wss.WithSubscriptions(subscriptions),
		wss.WithKeepalive(PingInterval, ping),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Bitmart").Err(err).Msg("error in bitmart.New")
		return nil, err
	}

	fetcher.Ws = ws
	return fetcher, nil
}

func (f *BitmartFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Bitmart").Err(err).Msg("error in bitmart.handleMessage")
		return err
	}
	if response.Table != "spot/ticker" {
		return nil
	}

	feedDataList := ResponseToFeedData(response, f.FeedMap)

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- feedData
	}

	return nil
}

func (f *BitmartFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}

// ping is the client keepalive, sent every PingInterval on each connection.
func ping(ctx context.Context, w wss.ConnWriter) error {
	log.Debug().Str("Player", "Bitmart").Msg("sending ping message to bitmart server")
	return w.RawWrite(ctx, "ping")
}

func (f *BitmartFetcher) customReadFunc(ctx context.Context, conn *websocket.Conn) (map[string]interface{}, error) {
	var result map[string]interface{}
	_, data, err := conn.Read(ctx)
	if err != nil {
		log.WithLevel(wss.ReadErrorLevel(ctx, zerolog.ErrorLevel)).Str("Player", "Bitmart").Err(err).Msg("error in Bitmart.customReadFunc, failed to read from websocket")
		return nil, err
	}
	if string(data) == "pong" {
		log.Debug().Str("Player", "Bitmart").Msg("received pong")
		return nil, nil
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Error().Str("Player", "Bitmart").Err(err).Msg("error in Bitmart.customReadFunc, failed to unmarshal data")
		return nil, err
	}

	return result, nil
}
