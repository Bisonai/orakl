package gemini

import (
	"context"
	"strings"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type GeminiFetcher struct {
	common.Fetcher
	quotes *common.QuoteCache

	bookMu sync.Mutex
	books  map[string]*topOfBook // symbol -> best bid/ask so far
}

type topOfBook struct{ bid, ask float64 }

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	config := &common.FetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fetcher := &GeminiFetcher{books: make(map[string]*topOfBook)}
	// REST 24h volumes let the book emit before the first trade (sparse trades)
	fetcher.quotes = common.NewQuoteCacheWithVolumes(&fetcher.VolumeCacheMap)
	fetcher.FeedMap = config.FeedMaps.Combined
	fetcher.FeedDataBuffer = config.FeedDataBuffer
	fetcher.VolumeCacheMap = common.VolumeCacheMap{
		Map:   make(map[int32]common.VolumeCache),
		Mutex: sync.Mutex{},
	}

	symbols := []string{}
	for feed := range fetcher.FeedMap {
		symbols = append(symbols, strings.ToUpper(feed))
	}

	ws, err := wss.NewWebsocketHelper(ctx,
		wss.WithEndpoint(URL+strings.Join(symbols, ",")),
		wss.WithSubscriptions([]any{}),
		wss.WithProxyUrl(config.Proxy))
	if err != nil {
		log.Error().Str("Player", "Gemini").Err(err).Msg("error in gemini.New")
		return nil, err
	}

	fetcher.Ws = ws
	return fetcher, nil
}

func (f *GeminiFetcher) handleMessage(ctx context.Context, message map[string]any) error {
	response, err := common.MessageToStruct[Response](message)
	if err != nil {
		log.Error().Str("Player", "Gemini").Err(err).Msg("error in MessageToResponse")
		return err
	}

	if response.Type != "update" || len(response.Events) == 0 {
		return nil
	}
	f.handleBookChanges(response.Events)

	feedDataList, err := TradeResponseToFeedDataList(response, f.FeedMap, &f.VolumeCacheMap)
	if err != nil {
		log.Error().Str("Player", "Gemini").Err(err).Msg("error in TradeResponseToFeedDataList")
		return err
	}
	for _, feedData := range feedDataList {
		f.FeedDataBuffer <- f.quotes.ApplyTrade(feedData)
	}
	return nil
}

// handleBookChanges applies top-of-book "change" events (one side each) and
// records the book of symbols whose both sides are known. The events carry
// no timestamp of their own, so the receive time is the book time.
func (f *GeminiFetcher) handleBookChanges(events []Event) {
	now := time.Now()
	changed := map[string]bool{}
	f.bookMu.Lock()
	for _, event := range events {
		if event.Type != "change" || event.Side == nil {
			continue
		}
		price, err := common.PriceStringToFloat64(event.Price)
		if err != nil {
			continue
		}
		book, ok := f.books[event.Symbol]
		if !ok {
			book = &topOfBook{}
			f.books[event.Symbol] = book
		}
		switch *event.Side {
		case "bid":
			book.bid = price
		case "ask":
			book.ask = price
		default:
			continue
		}
		changed[event.Symbol] = true
	}
	updates := map[string]topOfBook{}
	for symbol := range changed {
		if b := f.books[symbol]; b.bid > 0 && b.ask > 0 {
			updates[symbol] = *b
		}
	}
	f.bookMu.Unlock()

	for symbol, b := range updates {
		ids, exists := f.FeedMap[symbol]
		if !exists {
			continue
		}
		for _, feedData := range f.quotes.ApplyBook(ids, b.bid, b.ask, now) {
			f.FeedDataBuffer <- feedData
		}
	}
}

func (f *GeminiFetcher) Run(ctx context.Context) {
	go f.CacheVolumes(ctx)
	f.Ws.Run(ctx, f.handleMessage)
}

// volumeFetcher is overridable in tests.
var volumeFetcher = fetchVolumes

func (f *GeminiFetcher) CacheVolumes(ctx context.Context) {
	volumeTimer := time.NewTimer(common.VolumeFetchInterval * time.Millisecond)
	defer volumeTimer.Stop()

	if ctx.Err() != nil {
		return
	}
	volumeFetcher(ctx, f.FeedMap, &f.VolumeCacheMap)

	for {
		select {
		case <-ctx.Done():
			return
		case <-volumeTimer.C:
			if ctx.Err() != nil {
				return
			}
			volumeFetcher(ctx, f.FeedMap, &f.VolumeCacheMap)
			volumeTimer.Reset(common.VolumeFetchInterval * time.Millisecond)
		}
	}
}
