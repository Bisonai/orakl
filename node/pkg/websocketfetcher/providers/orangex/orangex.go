package orangex

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/utils/arr"
	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

const URL = "wss://api.orangex.com/ws/api/v1"

const (
	// OrangeX caps subscriptions at 30 channels per *connection* (not per
	// subscribe message): a second subscribe on a connection already holding 30
	// channels is rejected with code 1005 and
	// data {"limit":"30","requestedNew":"30","projected":"60","current":"30"}.
	// A request over the cap is rejected whole, so every chunk needs its own
	// connection.
	MaxChannelsPerConnection = 30

	// The server closes a connection ~30s after the last client message (error
	// 9901 "websocket closing connection automatically") even while tickers are
	// streaming; a websocket ping frame every 10s keeps it open. Ping frames
	// (not JSON-RPC /public/ping) so the replies don't reach the read loop:
	// otherwise they would reset the helper's inactivity timer and hide a
	// connection whose subscription was rejected.
	PingInterval = 10 * time.Second
	PingTimeout  = 5 * time.Second

	// ErrCodeAutoClose is sent right before the server drops an idle connection.
	ErrCodeAutoClose = 9901
)

type OrangeXFetcher struct {
	FeedMap        map[string][]int32
	FeedDataBuffer chan *common.FeedData
	Conns          []*wss.WebsocketHelper
}

type Params struct {
	Channels []string `json:"channels"`
}

type Subscription struct {
	JsonRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  Params `json:"params"`
	ID      int    `json:"id"`
}

// ErrorResponse is a JSON-RPC error reply, e.g. a rejected subscribe:
// {"id":"1","error":{"code":1005,"message":"Subscription channel limit exceeded","data":{...}}}
type ErrorResponse struct {
	ID    any `json:"id"`
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    any    `json:"data"`
	} `json:"error"`
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
			State          string  `json:"state"`
			LastPrice      string  `json:"last_price"`
			InstrumentName string  `json:"instrument_name"`
			MarkPrice      string  `json:"mark_price"`
			BestBidPrice   float64 `json:"best_bid_price"`
			BestBidAmount  float64 `json:"best_bid_amount"`
			BestAskPrice   float64 `json:"best_ask_price"`
			BestAskAmount  float64 `json:"best_ask_amount"`
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

	feeds := make([]string, 0, len(fetcher.FeedMap))
	for feed := range fetcher.FeedMap {
		feeds = append(feeds, feed)
	}

	subscriptions := BuildSubscriptions(feeds)
	for _, subscription := range subscriptions {
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
		fetcher.Conns = append(fetcher.Conns, ws)
	}

	return fetcher, nil
}

// BuildSubscriptions returns one subscribe request per connection, each with at
// most MaxChannelsPerConnection channels. Feeds are sorted so the split is
// deterministic across restarts.
func BuildSubscriptions(feeds []string) []Subscription {
	channels := make([]string, 0, len(feeds))
	for _, feed := range feeds {
		channels = append(channels, "ticker."+feed+".raw")
	}
	sort.Strings(channels)

	chunks := arr.SplitByChunkSize(channels, MaxChannelsPerConnection)
	subscriptions := make([]Subscription, 0, len(chunks))
	for i, chunk := range chunks {
		subscriptions = append(subscriptions, Subscription{
			JsonRPC: "2.0",
			Method:  "/public/subscribe",
			Params:  Params{Channels: chunk},
			ID:      i + 1,
		})
	}
	return subscriptions
}

// ParseErrorResponse reports whether message is a JSON-RPC error reply.
func ParseErrorResponse(message map[string]any) (ErrorResponse, bool) {
	if message["error"] == nil {
		return ErrorResponse{}, false
	}
	response, err := common.MessageToStruct[ErrorResponse](message)
	if err != nil {
		log.Error().Str("Player", "OrangeX").Err(err).Any("message", message).Msg("error parsing orangex error response")
		return ErrorResponse{}, true
	}
	return response, true
}

func (f *OrangeXFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	if errResponse, isError := ParseErrorResponse(message); isError {
		event := log.Error()
		if errResponse.Error.Code == ErrCodeAutoClose {
			event = log.Warn()
		}
		event.Str("Player", "OrangeX").
			Any("id", errResponse.ID).
			Int("code", errResponse.Error.Code).
			Str("message", errResponse.Error.Message).
			Any("data", errResponse.Error.Data).
			Msg("orangex returned an error response")
		return nil
	}

	raw, err := common.MessageToStruct[TickerResponse](message)
	if err != nil {
		log.Error().Str("Player", "OrangeX").Err(err).Msg("error in orangex.handleMessage")
		return err
	}

	if raw.Params.Data.InstrumentName == "" {
		return nil
	}

	feedDataList, err := TickerResponseToFeedData(raw, f.FeedMap)
	if err != nil {
		log.Error().Str("Player", "OrangeX").Err(err).Msg("error in orangex.handleMessage")
		return err
	}

	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- feedData
	}

	return nil
}

func (f *OrangeXFetcher) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, ws := range f.Conns {
		wg.Add(1)
		go func(ws *wss.WebsocketHelper) {
			defer wg.Done()
			go pingJob(ctx, ws)
			ws.Run(ctx, f.handleMessage)
		}(ws)
	}
	wg.Wait()
}

// pingJob keeps a connection from being auto-closed by the server. Ping
// errors are expected while the helper is reconnecting, so keep ticking; the
// per-ping timeout keeps a stalled connection from blocking the job.
func pingJob(ctx context.Context, ws *wss.WebsocketHelper) {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			conn := ws.Conn
			if conn == nil {
				continue
			}
			pingCtx, cancel := context.WithTimeout(ctx, PingTimeout)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				log.Debug().Str("Player", "OrangeX").Err(err).Msg("error in orangex.pingJob")
			}
		}
	}
}

func TickerResponseToFeedData(data TickerResponse, feedMap map[string][]int32) ([]*common.FeedData, error) {
	ids, exists := feedMap[data.Params.Data.InstrumentName]
	if !exists {
		log.Warn().Str("Player", "OrangeX").Any("data", data).Str("key", data.Params.Data.InstrumentName).Msg("feed not found")
		return nil, nil
	}

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

	result := []*common.FeedData{}
	for _, id := range ids {
		feedData := new(common.FeedData)
		feedData.FeedID = id
		feedData.Value = value
		feedData.Timestamp = &timestamp
		feedData.Volume = volume
		result = append(result, feedData)
	}

	return result, nil
}
