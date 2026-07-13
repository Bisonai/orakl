package mexc

import (
	"context"
	"encoding/json"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/mexc/pb"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"
)

type MexcFetcher common.Fetcher

// payloadKey hands the decoded protobuf wrapper from readFrame to handleMessage.
// The wss helper fixes the router signature at map[string]any, so the alternative
// would be re-encoding the wrapper to JSON purely to parse it straight back.
const payloadKey = "wrapper"

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &MexcFetcher{}
	fetcher.FeedMap = config.FeedMaps.Combined
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	subscription := Subscription{
		Method: "SUBSCRIPTION",
		Params: []string{Topic},
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions([]any{subscription}),
		wss.WithCustomReadFunc(readFrame),
		wss.WithReadLimit(ReadLimit),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Mexc").Err(err).Msg("error in mexc.New")
		return nil, err
	}
	fetcher.Ws = ws
	return fetcher, nil
}

// readFrame decodes MEXC's binary market-data frames. Subscription acks still
// arrive as text JSON on the same connection, so dispatch on the frame type.
//
// Neither branch returns an error for a bad frame: the wss helper tears the
// connection down and reconnects on anything the reader returns as an error, and
// one undecodable frame is not worth a reconnect.
func readFrame(ctx context.Context, conn *websocket.Conn) (map[string]any, error) {
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		return nil, err
	}

	if messageType != websocket.MessageBinary {
		var ack Ack
		if err := json.Unmarshal(data, &ack); err != nil {
			log.Warn().Str("Player", "Mexc").Str("frame", string(data)).Msg("unrecognized text frame")
			return nil, nil
		}

		// A successful ack echoes the topic back. Anything else means this
		// connection will never carry data -- including a retired topic, which MEXC
		// reports as `{"code":0,"msg":"Not Subscribed successfully! ... Blocked! "}`.
		// Log it, rather than leaving the connection open and silent as before.
		if ack.Code != 0 || ack.Msg != Topic {
			log.Error().Str("Player", "Mexc").
				Int("code", ack.Code).
				Str("msg", ack.Msg).
				Msg("mexc rejected the subscription")
		}
		return nil, nil
	}

	wrapper := &pb.PushDataV3ApiWrapper{}
	if err := proto.Unmarshal(data, wrapper); err != nil {
		log.Error().Str("Player", "Mexc").Err(err).Msg("failed to unmarshal protobuf frame")
		return nil, nil
	}

	return map[string]any{payloadKey: wrapper}, nil
}

func (f *MexcFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	wrapper, ok := message[payloadKey].(*pb.PushDataV3ApiWrapper)
	if !ok {
		return nil
	}

	feedDataList, err := WrapperToFeedDataList(wrapper, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "Mexc").Err(err).Msg("failed to extract feedData from message")
		return err
	}

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- feedData
	}

	return nil
}

func (f *MexcFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}
