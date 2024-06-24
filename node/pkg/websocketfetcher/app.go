package websocketfetcher

import (
	"context"
	"errors"
	"os"
	"time"

	"bisonai.com/orakl/node/pkg/chain/websocketchainreader"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/binance"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/bingx"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/bitget"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/bithumb"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/bitmart"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/bitstamp"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/btse"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/bybit"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/coinbase"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/coinex"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/coinone"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/crypto"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/gateio"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/gemini"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/huobi"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/korbit"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/kraken"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/kucoin"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/lbank"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/mexc"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/okx"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/uniswap"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/upbit"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/xt"
	"github.com/rs/zerolog/log"
)

// TODO: utilize unused providers: bitstamp, gemini, lbank, bitget (should be included in orakl config)
const (
	DefaultStoreInterval = 500 * time.Millisecond
	DefaultBufferSize    = 500
)

type AppConfig struct {
	SetFromDB     bool
	Feeds         []common.Feed
	CexFactories  map[string]func(context.Context, ...common.FetcherOption) (common.FetcherInterface, error)
	DexFactories  map[string]func(...common.DexFetcherOption) common.FetcherInterface
	BufferSize    int
	StoreInterval time.Duration
}

type AppOption func(*AppConfig)

func WithSetFromDB(useDB bool) AppOption {
	return func(c *AppConfig) {
		c.SetFromDB = useDB
	}
}

func WithFeeds(feeds []common.Feed) AppOption {
	return func(c *AppConfig) {
		c.Feeds = feeds
	}
}

func WithCexFactories(factories map[string]func(context.Context, ...common.FetcherOption) (common.FetcherInterface, error)) AppOption {
	return func(c *AppConfig) {
		c.CexFactories = factories
	}
}

func WithBufferSize(size int) AppOption {
	return func(c *AppConfig) {
		c.BufferSize = size
	}
}

func WithStoreInterval(interval time.Duration) AppOption {
	return func(c *AppConfig) {
		c.StoreInterval = interval
	}
}

type App struct {
	fetchers      []common.FetcherInterface
	buffer        chan common.FeedData
	storeInterval time.Duration
	chainReader   *websocketchainreader.ChainReader
}

func New() *App {
	return &App{}
}

func (a *App) Init(ctx context.Context, opts ...AppOption) error {
	// TODO: Proxy support
	cexFactories := map[string]func(context.Context, ...common.FetcherOption) (common.FetcherInterface, error){
		"binance":  binance.New,
		"coinbase": coinbase.New,
		"coinone":  coinone.New,
		"korbit":   korbit.New,
		"kucoin":   kucoin.New,
		"bybit":    bybit.New,
		"upbit":    upbit.New,
		"crypto":   crypto.New,
		"btse":     btse.New,
		"bithumb":  bithumb.New,
		"gateio":   gateio.New,
		"coinex":   coinex.New,
		"huobi":    huobi.New,
		"mexc":     mexc.New,
		"bitstamp": bitstamp.New,
		"gemini":   gemini.New,
		"lbank":    lbank.New,
		"bitget":   bitget.New,
		"okx":      okx.New,
		"kraken":   kraken.New,
		"bingx":    bingx.New,
		"bitmart":  bitmart.New,
		"xt":       xt.New,
	}

	dexFactories := map[string]func(...common.DexFetcherOption) common.FetcherInterface{
		"uniswap": uniswap.New,
	}

	appConfig := &AppConfig{
		SetFromDB:     true,
		CexFactories:  cexFactories,
		DexFactories:  dexFactories,
		BufferSize:    DefaultBufferSize,
		StoreInterval: DefaultStoreInterval,
	}

	for _, opt := range opts {
		opt(appConfig)
	}

	if err := a.initializeCex(ctx, *appConfig); err != nil {
		return err
	}

	if err := a.initializeDex(ctx, *appConfig); err != nil {
		return err
	}

	return nil
}

func (a *App) initializeCex(ctx context.Context, appConfig AppConfig) error {
	var feeds []common.Feed
	if appConfig.SetFromDB {
		var err error
		feeds, err = db.QueryRows[common.Feed](ctx, common.GetAllWebsocketFeedsQuery, nil)
		if err != nil {
			log.Error().Err(err).Msg("error in fetching feeds")
			return err
		}
	}

	if len(appConfig.Feeds) > 0 {
		feeds = appConfig.Feeds
	}
	feedMap := common.GetWssFeedMap(feeds)

	a.buffer = make(chan common.FeedData, appConfig.BufferSize)
	a.storeInterval = appConfig.StoreInterval

	for name, factory := range appConfig.CexFactories {
		if _, ok := feedMap[name]; !ok {
			log.Warn().Msgf("no feeds for %s", name)
			continue
		}
		fetcher, err := factory(
			ctx,
			common.WithFeedDataBuffer(a.buffer),
			common.WithFeedMaps(feedMap[name]),
		)
		if err != nil {
			log.Error().Err(err).Msgf("error in creating %s fetcher", name)
			return err
		}
		a.fetchers = append(a.fetchers, fetcher)
	}
	return nil
}

func (a *App) initializeDex(ctx context.Context, appConfig AppConfig) error {
	kaiaWebsocketUrl := os.Getenv("KAIA_WEBSOCKET_URL")
	ethWebsocketUrl := os.Getenv("ETH_WEBSOCKET_URL")
	if kaiaWebsocketUrl == "" || ethWebsocketUrl == "" {
		log.Error().Msg("KAIA_WEBSOCKET_URL and ETH_WEBSOCKET_URL must be set")
		return errors.New("KAIA_WEBSOCKET_URL and ETH_WEBSOCKET_URL must be set")
	}

	chainReader, err := websocketchainreader.New(kaiaWebsocketUrl, ethWebsocketUrl)
	if err != nil {
		log.Error().Err(err).Msg("error in creating chain reader")
		return err
	}
	a.chainReader = chainReader

	for name, factory := range appConfig.DexFactories {
		feeds, err := db.QueryRows[common.Feed](ctx, common.GetDexFeedsQuery(name), nil)
		if err != nil {
			log.Error().Err(err).Msg("error in fetching feeds")
			return err
		}
		fetcher := factory(
			common.WithFeeds(feeds),
			common.WithDexFeedDataBuffer(a.buffer),
			common.WithWebsocketChainReader(a.chainReader),
		)

		a.fetchers = append(a.fetchers, fetcher)
	}

	return nil
}

func (a *App) Start(ctx context.Context) {
	for _, fetcher := range a.fetchers {
		go fetcher.Run(ctx)
	}

	ticker := time.NewTicker(a.storeInterval)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			a.storeFeedData(ctx)
		}
	}
}

func (a *App) storeFeedData(ctx context.Context) {

	select {
	case <-ctx.Done():
		return
	case feedData := <-a.buffer:
		batch := []common.FeedData{feedData}
		// Continue to drain the buffer until it's empty
	loop:
		for {
			select {
			case feedData := <-a.buffer:
				batch = append(batch, feedData)
			default:
				break loop
			}
		}

		err := common.StoreFeeds(ctx, batch)
		if err != nil {
			log.Error().Err(err).Msg("error in storing feed data")
		}
	default:
		return
	}
}
