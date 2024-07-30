package gopax

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

type GopaxFetcher common.Fetcher

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &GopaxFetcher{}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	subscription := []any{
		Subscription{
			Name: "SubscribeToTickers",
		},
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions(subscription),
		wss.WithProxyUrl(config.Proxy),
		wss.WithReadLimit(IncreasedReadLimit),
		wss.WithCustomReadFunc(fetcher.customReadFunc),
	)
	if err != nil {
		log.Error().Str("Player", "Gopax").Err(err).Msg("error in gopax.New")
		return nil, err
	}
	fetcher.Ws = ws
	return fetcher, nil
}

func (f *GopaxFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		return err
	}

	if response.Name == "SubscribeToTickers" {
		var initialResponse InitialResponse
		err := json.Unmarshal(response.Object, &initialResponse)
		if err != nil {
			return fmt.Errorf("failed to parse initial response: %v", response)
		}
		feedDataList := InitialResponseToFeedData(initialResponse, f.FeedMap)

		for _, feedData := range feedDataList {
			f.FeedDataBuffer <- *feedData
		}
		return nil
	}

	if response.Name == "TickerEvent" {
		var tickers Tickers
		err := json.Unmarshal(response.Object, &tickers)
		if err != nil {
			return fmt.Errorf("failed to parse response: %v", response)
		}
		for _, ticker := range tickers {
			feedData, err := TickerToFeedData(ticker, f.FeedMap)
			if err != nil {
				if err.Error() == "Feed not found" {
					continue
				}
				log.Error().Err(err).Str("Player", "Gopax").Msg("error in gopax.handleMessage, failed to convert ticker to feed data")
				continue
			}
			f.FeedDataBuffer <- *feedData
		}

		return nil
	}
	return nil
}

func (f *GopaxFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}

func (f *GopaxFetcher) customReadFunc(ctx context.Context, conn *websocket.Conn) (map[string]interface{}, error) {
	var result map[string]interface{}
	_, data, err := conn.Read(ctx)
	if err != nil {
		log.Error().Str("Player", "Gopax").Err(err).Msg("error in gopax.customReadFunc, failed to read from websocket")
		return nil, err
	}

	rawResponse := string(data)

	if strings.HasPrefix(rawResponse, "\"primus::ping::") {
		log.Debug().Str("Player", "Gopax").Msg("received pong")
		_ = f.Ws.RawWrite(ctx, "\"primus::pong::"+rawResponse[15:])

		return nil, nil
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Error().Str("Player", "Gopax").Err(err).Msg("error in gopax.customReadFunc, failed to unmarshal data")
		return nil, err
	}

	return result, nil
}
