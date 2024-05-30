package websocketfetcher

import (
	"context"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/binance"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/coinbase"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/coinone"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/korbit"
	"github.com/rs/zerolog/log"
)

const (
	DefaultStoreInterval = 1000 * time.Millisecond
	DefaultBufferSize    = 500
)

type AppConfig struct {
	SetFromDB     bool
	Feeds         []common.Feed
	Factories     map[string]func(context.Context, ...common.FetcherOption) (common.FetcherInterface, error)
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

func WithFactories(factories map[string]func(context.Context, ...common.FetcherOption) (common.FetcherInterface, error)) AppOption {
	return func(c *AppConfig) {
		c.Factories = factories
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
}

func New() *App {
	return &App{}
}

func (a *App) Init(ctx context.Context, opts ...AppOption) error {
	// TODO: Proxy support
	factories := map[string]func(context.Context, ...common.FetcherOption) (common.FetcherInterface, error){
		"binance":  binance.New,
		"coinbase": coinbase.New,
		"coinone":  coinone.New,
		"korbit":   korbit.New,
	}

	appConfig := &AppConfig{
		SetFromDB:     true,
		Factories:     factories,
		BufferSize:    DefaultBufferSize,
		StoreInterval: DefaultStoreInterval,
	}

	for _, opt := range opts {
		opt(appConfig)
	}

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

	for name, creator := range appConfig.Factories {
		fetcher, err := creator(
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
		draining := true
		for draining {
			select {
			case feedData := <-a.buffer:
				batch = append(batch, feedData)
			default:
				draining = false
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
