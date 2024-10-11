package orangex

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

const URL = "wss://api.orangex.com/ws/api/v1"

type OrangeXFetcher common.Fetcher

type Params struct {
	Channels []string `json:"channels"`
}

type Subscription struct {
	JsonRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  Params `json:"params"`
	ID      int    `json:"id"`
}

type TickerResponse struct {
	Params struct {
		Data struct {
			Timestamp string `json:"timestamp"`
			Stats     struct {
				Volume      string `json:"volume"`
				PriceChange string `json:"price_change"`
				Low         string `json:"low"`
				Turnover    string `json:"turnover"`
				High        string `json:"high"`
			} `json:"stats"`
			State          string `json:"state"`
			LastPrice      string `json:"last_price"`
			InstrumentName string `json:"instrument_name"`
			MarkPrice      string `json:"mark_price"`
			BestBidPrice   string `json:"best_bid_price"`
			BestBidAmount  string `json:"best_bid_amount"`
			BestAskPrice   string `json:"best_ask_price"`
			BestAskAmount  string `json:"best_ask_amount"`
		} `json:"data"`
		Channel string `json:"channel"`
	} `json:"params"`
	Method  string `json:"method"`
	JSONRPC string `json:"jsonrpc"`
}

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &OrangeXFetcher{}
	fetcher.FeedMap = config.FeedMaps.Separated
	fetcher.FeedDataBuffer = config.FeedDataBuffer

	channels := []string{}
	for feed := range fetcher.FeedMap {
		channels = append(channels, "ticker."+feed+".raw")
	}

	params := Params{Channels: channels}

	subscription := Subscription{
		JsonRPC: "2.0",
		Method:  "/public/subscribe",
		Params:  params,
		ID:      1,
	}

	log.Debug().Any("subscription", subscription).Msg("subscription generated")

	// since wsjson.Write didn't work for orangex, had to pass marshalled byte instead
	raw, err := json.Marshal(subscription)
	if err != nil {
		log.Error().Str("Player", "OrangeX").Err(err).Msg("error in orangex.New")
		return nil, err
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL),
		wss.WithSubscriptions([]any{raw}),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "OrangeX").Err(err).Msg("error in orangex.New")
		return nil, err
	}
	fetcher.Ws = ws
	return fetcher, nil
}

func (f *OrangeXFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	raw, err := common.MessageToStruct[TickerResponse](message)
	if err != nil {
		log.Error().Str("Player", "OrangeX").Err(err).Msg("error in orangex.handleMessage")
		return err
	}

	if raw.Params.Data.InstrumentName == "" {
		return nil
	}

	feedData, err := TickerResponseToFeedData(raw, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "OrangeX").Err(err).Msg("error in orangex.handleMessage")
		return err
	}

	if feedData == nil {
		return nil
	}

	f.FeedDataBuffer <- feedData
	return nil
}

func (f *OrangeXFetcher) Run(ctx context.Context) {
	f.Ws.Run(ctx, f.handleMessage)
}

func TickerResponseToFeedData(data TickerResponse, feedMap map[string]int32) (*common.FeedData, error) {
	id, exists := feedMap[data.Params.Data.InstrumentName]
	if !exists {
		log.Warn().Str("Player", "OrangeX").Any("data", data).Str("key", data.Params.Data.InstrumentName).Msg("feed not found")
		return nil, nil
	}

	feedData := new(common.FeedData)
	value, err := common.PriceStringToFloat64(data.Params.Data.LastPrice)
	if err != nil {
		log.Error().Str("Player", "OrangeX").Err(err).Msg("error in PriceStringToFloat64")
		return nil, err
	}

	intTimestamp, err := strconv.ParseInt(data.Params.Data.Timestamp, 10, 64)
	if err != nil {
		log.Error().Str("Player", "OrangeX").Err(err).Msg("error in strconv.ParseInt")
		return nil, err
	}
	timestamp := time.UnixMilli(intTimestamp)
	volume, err := common.VolumeStringToFloat64(data.Params.Data.Stats.Volume)
	if err != nil {
		log.Error().Str("Player", "OrangeX").Err(err).Msg("error in VolumeStringToFloat64")
		return nil, err
	}

	feedData.FeedID = id
	feedData.Value = value
	feedData.Timestamp = &timestamp
	feedData.Volume = volume
	return feedData, nil
}
