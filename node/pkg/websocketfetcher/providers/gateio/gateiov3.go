package gateio

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/utils/arr"
	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

const v3Url = "wss://ws.gate.io/v3/"

type GateioV3Fetcher common.Fetcher

type v3Subscription struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

type TickerUpdate struct {
	Method string       `json:"method"`
	Params TickerParams `json:"params"`
}

type TickerParams [2]interface{}

type TickerData struct {
	Last       string `json:"last"`
	BaseVolume string `json:"baseVolume"`
}

func NewV3(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &GateioV3Fetcher{}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	payload := []string{}
	for feed := range fetcher.FeedMap {
		payload = append(payload, strings.ReplaceAll(feed, "-", "_"))
	}

	maxBatchSize := 50
	splittedPayloads := arr.SplitByChunkSize(payload, maxBatchSize)

	channel := "ticker.subscribe"

	subscriptions := make([]any, 0, len(splittedPayloads))
	for _, sp := range splittedPayloads {
		subscriptions = append(subscriptions, v3Subscription{
			Method: channel,
			Params: sp,
		})
	}

	log.Debug().Any("subscriptions", subscriptions).Msg("subscriptions generated")

	ws, err := wss.NewWebsocketHelper(
		ctx,
		wss.WithEndpoint(v3Url),
		wss.WithSubscriptions(subscriptions),
		wss.WithProxyUrl(config.Proxy),
	)
	if err != nil {
		return nil, err
	}

	fetcher.Ws = ws
	return fetcher, nil
}

func (f *GateioV3Fetcher) handleMessage(ctx context.Context, message map[string]any) error {
	response, err := common.MessageToStruct[TickerUpdate](message)
	if err != nil {
		log.Error().Str("Player", "GateioV3").Err(err).Msg("failed to unmarshal message")
		return err
	}

	if response.Method != "ticker.update" {
		return nil
	}

	// Extract the params into proper types
	symbol := response.Params[0].(string)
	key := strings.Replace(symbol, "_", "-", 1)
	ids, exists := f.FeedMap[key]
	if !exists {
		return nil
	}

	// Marshal the second param into TickerData struct
	tickerBytes, _ := json.Marshal(response.Params[1])
	var ticker TickerData
	_ = json.Unmarshal(tickerBytes, &ticker)

	timestamp := time.Now().UTC()

	price, err := common.PriceStringToFloat64(ticker.Last)
	if err != nil {
		return nil
	}

	volume, err := common.VolumeStringToFloat64(ticker.BaseVolume)
	if err != nil {
		return nil
	}

	for _, id := range ids {
		f.FeedDataBuffer <- &common.FeedData{
			FeedID:    id,
			Value:     price,
			Volume:    volume,
			Timestamp: &timestamp,
		}
	}
	return nil

}

func (f *GateioV3Fetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}
