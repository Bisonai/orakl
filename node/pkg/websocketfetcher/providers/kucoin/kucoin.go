package kucoin

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

	"net/http"

	"bisonai.com/miko/node/pkg/utils/request"
	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
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

	symbols := []string{}

	for feed := range fetcher.FeedMap {
		symbols = append(symbols, feed)
	}

	// Kucoin rejects a subscribe frame carrying more than MaxSymbolsPerSubscription
	// symbols with `{"type":"error","code":509,"data":"exceed max subscription
	// count limitation of 100 per time"}`, which drops every symbol in the frame --
	// not just the overflow. Split the symbols across several subscribe frames; the
	// wss helper sends them sequentially on (re)connect.
	subscriptions := []any{}
	for i := 0; i < len(symbols); i += MaxSymbolsPerSubscription {
		end := min(i+MaxSymbolsPerSubscription, len(symbols))
		subscriptions = append(subscriptions, Subscription{
			ID:       i/MaxSymbolsPerSubscription + 1,
			Type:     "subscribe",
			Topic:    "/market/snapshot:" + strings.Join(symbols[i:end], ","),
			Response: true,
		})
	}

	ws, err := newWebsocketHelper(ctx, TokenUrl,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions(subscriptions),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Kucoin").Err(err).Msg("error in kucoin.New")
		return nil, err
	}
	fetcher.Ws = ws
	return fetcher, nil
}

func (f *KucoinFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	// Kucoin interleaves control frames (welcome / ack / pong) and error frames
	// with market data. Error frames carry `data` as a string, so unmarshalling
	// one into SymbolSnapshotRaw -- whose Data is a struct -- fails, and the
	// actual reason (e.g. "exceed max subscription count limitation of 100 per
	// time") is lost behind a generic unmarshal error. Dispatch on type first.
	switch messageType, _ := message["type"].(string); messageType {
	case "message":
		// market data; parsed below
	case "error":
		log.Error().Str("Player", "Kucoin").
			Any("code", message["code"]).
			Any("data", message["data"]).
			Msg("kucoin rejected a subscription")
		return nil
	default:
		return nil
	}

	raw, err := common.MessageToStruct[SymbolSnapshotRaw](message)
	if err != nil {
		log.Error().Str("Player", "Kucoin").Err(err).Msg("error in kucoin.handleMessage")
		return err
	}

	if raw.Subject != "trade.snapshot" {
		return nil
	}

	feedDataList, err := RawDataToFeedData(raw, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Kucoin").Err(err).Msg("error in kucoin.handleMessage")
		return err
	}

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- feedData
	}

	return nil
}

func (f *KucoinFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}

// newWebsocketHelper wires the bullet-API dial and the client ping. The ping
// interval comes from the bullet API on every dial and is applied to that
// connection; it is per helper, so fetchers do not share it.
func newWebsocketHelper(ctx context.Context, tokenURL string, opts ...wss.ConnectionOption) (*wss.WebsocketHelper, error) {
	var pingIntervalMs atomic.Int64
	pingIntervalMs.Store(DEFAULT_PING_INTERVAL)

	opts = append(opts,
		wss.WithCustomDialFunc(newDialFunc(tokenURL, &pingIntervalMs)),
		wss.WithKeepalive(DEFAULT_PING_INTERVAL*time.Millisecond, ping),
		wss.WithKeepaliveIntervalFunc(func() time.Duration {
			return time.Duration(pingIntervalMs.Load()) * time.Millisecond
		}),
	)
	return wss.NewWebsocketHelper(ctx, opts...)
}

// ping is the client keepalive.
func ping(ctx context.Context, w wss.ConnWriter) error {
	log.Debug().Msg("sending ping message to kucoin server")
	return w.Write(ctx, Ping{
		ID:   1,
		Type: "ping",
	})
}

// newDialFunc fetches a token and ping interval from the bullet API before
// each dial and stores the interval (validated, else the default) in
// pingIntervalMs for the connection being dialed.
func newDialFunc(tokenURL string, pingIntervalMs *atomic.Int64) func(context.Context, string, *websocket.DialOptions) (*websocket.Conn, *http.Response, error) {
	return func(ctx context.Context, endpoint string, dialOptions *websocket.DialOptions) (*websocket.Conn, *http.Response, error) {
		token, interval, err := getTokenAndPingInterval(tokenURL)
		if err != nil {
			log.Error().Str("Player", "Kucoin").Err(err).Msg("error in kucoin.customDialFunc")
			return nil, nil, err
		}
		if interval <= 0 {
			log.Warn().Str("Player", "Kucoin").Int("pingInterval", interval).Msg("invalid kucoin ping interval, using default")
			interval = DEFAULT_PING_INTERVAL
		}
		pingIntervalMs.Store(int64(interval))
		log.Debug().Int("pingInterval", interval).Msg("kucoin ping interval set")

		url := endpoint + "?token=" + token

		return websocket.Dial(ctx, url, dialOptions)
	}
}

// getTokenAndPingInterval returns the token and the ping interval in ms; the
// interval is 0 when the response has no instance server.
func getTokenAndPingInterval(tokenURL string) (string, int, error) {
	resp, err := request.Request[TokenResponse](request.WithEndpoint(tokenURL), request.WithMethod("POST"))
	if err != nil {
		log.Error().Str("Player", "Kucoin").Err(err).Msg("error in kucoin.getToken")
		return "", DEFAULT_PING_INTERVAL, err
	}
	if len(resp.Data.InstanceServers) == 0 {
		return resp.Data.Token, 0, nil
	}
	return resp.Data.Token, resp.Data.InstanceServers[0].PingInterval, nil
}
