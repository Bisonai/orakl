package kucoin

import (
	"context"
	"strings"
	"time"

	"net/http"

	"bisonai.com/orakl/node/pkg/utils/request"
	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

// TODO: to retrieve volume, use `marketSnapshot api` instead of ticker

type KucoinFetcher common.Fetcher

var pingInterval = DEFAULT_PING_INTERVAL

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &KucoinFetcher{}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	symbols := []string{}
	for feed := range fetcher.FeedMap {
		symbols = append(symbols, feed)
	}

	subscription := Subscription{
		ID:       1,
		Type:     "subscribe",
		Topic:    "/market/ticker:" + strings.Join(symbols, ","),
		Response: true,
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithCustomDialFunc(fetcher.customDialFunc),
		wss.WithEndpoint(URL),
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
	go f.pingJob(ctx)
	f.Ws.Run(ctx, f.handleMessage)
}

func (f *KucoinFetcher) pingJob(ctx context.Context) {
	pingTicker := time.NewTicker(time.Duration(pingInterval) * time.Millisecond)

	for range pingTicker.C {
		log.Debug().Msg("sending ping message to kucoin server")
		err := f.Ws.Write(ctx, Ping{
			ID:   1,
			Type: "ping",
		})
		if err != nil {
			log.Error().Str("Player", "Kucoin").Err(err).Msg("error in kucoin.pingJob")
			return
		}
	}
}

func (f *KucoinFetcher) customDialFunc(ctx context.Context, endpoint string, dialOptions *websocket.DialOptions) (*websocket.Conn, *http.Response, error) {
	token, interval, err := f.getTokenAndPingInterval()
	if err != nil {
		log.Error().Str("Player", "Kucoin").Err(err).Msg("error in kucoin.customDialFunc")
		return nil, nil, err
	}
	pingInterval = interval
	log.Debug().Int("pingInterval", interval).Msg("kucoin ping interval set")

	url := endpoint + "?token=" + token

	return websocket.Dial(ctx, url, dialOptions)
}

func (f *KucoinFetcher) getTokenAndPingInterval() (string, int, error) {
	resp, err := request.UrlRequest[TokenResponse](TokenUrl, "POST", nil, nil, "")
	if err != nil {
		log.Error().Str("Player", "Kucoin").Err(err).Msg("error in kucoin.getToken")
		return "", 18000, err
	}
	return resp.Data.Token, resp.Data.InstanceServers[0].PingInterval, nil
}
