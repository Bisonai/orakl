package websocketfetcher

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/chain/websocketchainreader"
	"bisonai.com/miko/node/pkg/common/types"
	"bisonai.com/miko/node/pkg/db"
	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/binance"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/bingx"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/bitget"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/bithumb"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/bitmart"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/bitstamp"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/btse"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/bybit"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/coinbase"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/coinex"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/coinone"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/crypto"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/gateio"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/gemini"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/gopax"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/huobi"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/korbit"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/kraken"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/kucoin"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/lbank"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/mexc"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/okx"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/orangex"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/uniswap"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/upbit"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/xt"
	"github.com/rs/zerolog/log"
)

const (
	DefaultStoreInterval = 200 * time.Millisecond
	DefaultBufferSize    = 3000
)

type AppConfig struct {
	SetFromDB           bool
	Feeds               []common.Feed
	CexFactories        map[string]func(context.Context, ...common.FetcherOption) (common.FetcherInterface, error)
	DexFactories        map[string]func(...common.DexFetcherOption) common.FetcherInterface
	BufferSize          int
	StoreInterval       time.Duration
	LatestFeedDataMap   *types.LatestFeedDataMap
	FeedDataDumpChannel chan *types.FeedData
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

func WithLatestFeedDataMap(latestFeedDataMap *types.LatestFeedDataMap) AppOption {
	return func(c *AppConfig) {
		c.LatestFeedDataMap = latestFeedDataMap
	}
}

func WithFeedDataDumpChannel(feedDataDumpChannel chan *types.FeedData) AppOption {
	return func(c *AppConfig) {
		c.FeedDataDumpChannel = feedDataDumpChannel
	}
}

type App struct {
	fetchers            []common.FetcherInterface
	buffer              chan *common.FeedData
	storeInterval       time.Duration
	chainReader         *websocketchainreader.ChainReader
	latestFeedDataMap   *types.LatestFeedDataMap
	feedDataDumpChannel chan *common.FeedData
	cancel              context.CancelFunc
}

func New() *App {
	return &App{}
}

func (a *App) Init(ctx context.Context, opts ...AppOption) error {

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
		"gopax":    gopax.New,
		"orangex":  orangex.New,
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
		LatestFeedDataMap: &types.LatestFeedDataMap{
			FeedDataMap: make(map[int32]*types.FeedData),
			Mu:          sync.RWMutex{},
		},
		FeedDataDumpChannel: make(chan *types.FeedData, 10000),
	}

	for _, opt := range opts {
		opt(appConfig)
	}

	a.latestFeedDataMap = appConfig.LatestFeedDataMap
	a.feedDataDumpChannel = appConfig.FeedDataDumpChannel

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

	a.buffer = make(chan *common.FeedData, appConfig.BufferSize)
	a.storeInterval = appConfig.StoreInterval

	wsProxy := os.Getenv("WS_PROXY")

	for name, factory := range appConfig.CexFactories {
		if _, ok := feedMap[name]; !ok {
			log.Warn().Msgf("no feeds for %s", name)
			continue
		}
		fetcher, err := factory(
			ctx,
			common.WithFeedDataBuffer(a.buffer),
			common.WithFeedMaps(feedMap[name]),
			common.WithProxy(wsProxy),
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

	chainReader, err := websocketchainreader.New(
		websocketchainreader.WithEthWebsocketUrl(ethWebsocketUrl),
		websocketchainreader.WithKaiaWebsocketUrl(kaiaWebsocketUrl))
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
	ctxWithCancel, cancel := context.WithCancel(ctx)
	a.cancel = cancel

	for _, fetcher := range a.fetchers {
		go fetcher.Run(ctxWithCancel)
	}

	ticker := time.NewTicker(a.storeInterval)
	for {
		select {
		case <-ctxWithCancel.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			go a.storeFeedData(ctxWithCancel)
		}
	}
}

func (a *App) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
}

func (a *App) storeFeedData(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case feedData := <-a.buffer:
		batch := []*common.FeedData{feedData}
		// Continue to drain the buffer until it's empty
	loop:
		for {
			select {
			case feedData := <-a.buffer:
				batch = append(batch, feedData)
				a.feedDataDumpChannel <- feedData
			default:
				break loop
			}
		}

		err := a.latestFeedDataMap.SetLatestFeedData(batch)
		if err != nil {
			log.Error().Err(err).Msg("error in setting latest feed data")
		}
	default:
		return
	}
}
