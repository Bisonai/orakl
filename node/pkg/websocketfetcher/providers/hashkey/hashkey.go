package hashkey

import (
	"context"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type HashkeyFetcher common.Fetcher

// expected to receive a feedmap with keys in "<base><quote>" upper-case form
func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &HashkeyFetcher{}
	fetcher.FeedMap = config.FeedMaps.Combined
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	// HashKey accepts one symbol per subscribe request, so send one per feed.
	subscriptions := []any{}
	for feed := range fetcher.FeedMap {
		subscriptions = append(subscriptions, Subscription{
			Topic:  "realtimes",
			Event:  "sub",
			Params: map[string]string{"symbol": feed},
		})
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions(subscriptions),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Hashkey").Err(err).Msg("error in hashkey.New")
		return nil, err
	}
	fetcher.Ws = ws

	return fetcher, nil
}

func (f *HashkeyFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	// Subscription acks carry an "event" field; a non-zero code means the sub
	// was rejected. Log it loudly -- a silently rejected sub is a feed that
	// never produces data.
	if _, isAck := message["event"]; isAck {
		if code, _ := message["code"].(string); code != "" && code != "0" {
			log.Error().Str("Player", "Hashkey").
				Any("code", message["code"]).
				Any("msg", message["msg"]).
				Any("params", message["params"]).
				Msg("hashkey rejected a subscription")
		}
		return nil
	}

	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Hashkey").Err(err).Msg("error in hashkey.handleMessage")
		return err
	}

	// Skips the server's pong frames and anything that isn't a ticker push.
	if response.Topic != "realtimes" || response.Data.Symbol == "" {
		return nil
	}

	feedDataList, err := ResponseToFeedDataList(response, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Hashkey").Err(err).Msg("error in hashkey.handleMessage")
		return err
	}

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- feedData
	}

	return nil
}

func (f *HashkeyFetcher) Run(ctx context.Context) {
	go f.pingJob(ctx)
	f.Ws.Run(ctx, f.handleMessage)
}

// pingJob is the client keep-alive. HashKey never pings us and drops a silent
// connection after ~30s; a `{"ping":<ms>}` every 10s keeps it open (pingJob is
// the only writer, so this does not race with the read loop).
func (f *HashkeyFetcher) pingJob(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(PingInterval) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := f.Ws.Write(ctx, Ping{Ping: time.Now().UnixMilli()}); err != nil {
				log.Error().Str("Player", "Hashkey").Err(err).Msg("error in hashkey.pingJob")
				return
			}
		}
	}
}
