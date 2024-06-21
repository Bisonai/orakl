package bitget

import (
	"context"
	"encoding/json"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

type BitgetFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &BitgetFetcher{}
	fetcher.FeedMap = config.FeedMaps.Combined
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	args := []Arg{}
	for feed := range fetcher.FeedMap {
		arg := Arg{
			InstType: "SP",
			Channel:  "ticker",
			InstId:   feed,
		}
		args = append(args, arg)
	}
	subscription := Subscription{
		Op:   "subscribe",
		Args: args,
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithCustomReadFunc(fetcher.customReadFunc),
		wss.WithEndpoint(URL),
		wss.WithSubscriptions([]any{subscription}),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Bitget").Err(err).Msg("error in bitget.New")
		return nil, err
	}

	fetcher.Ws = ws
	return fetcher, nil
}

func (f *BitgetFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Bitget").Err(err).Msg("error in MessageToResponse")
		return err
	}
	feedDataList, err := ResponseToFeedDataList(response, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Bitget").Err(err).Msg("error in ResponseToFeedDataList")
		return err
	}
	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- *feedData
	}

	return nil
}

func (f *BitgetFetcher) Run(ctx context.Context) {
	f.ping(ctx)
	f.Ws.Run(ctx, f.handleMessage)
}

func (f *BitgetFetcher) ping(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				log.Debug().Str("Player", "Bitget").Msg("sending ping message to bitget server")
				err := f.Ws.RawWrite(ctx, "ping")
				if err != nil {
					log.Error().Str("Player", "Bitget").Err(err).Msg("error in bitget.ping")
				}
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

func (f *BitgetFetcher) customReadFunc(ctx context.Context, conn *websocket.Conn) (map[string]interface{}, error) {
	var result map[string]interface{}
	_, data, err := conn.Read(ctx)
	if err != nil {
		log.Error().Str("Player", "Bitget").Err(err).Msg("error in Bitget.customReadFunc, failed to read from websocket")
		return nil, err
	}
	if string(data) == "pong" {
		log.Debug().Str("Player", "Bitget").Msg("received pong")
		return nil, nil
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Error().Str("Player", "Bitget").Err(err).Msg("error in Bitget.customReadFunc, failed to unmarshal data")
		return nil, err
	}

	return result, nil
}
