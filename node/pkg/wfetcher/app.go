package wfetcher

import (
	"context"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/wfetcher/common"
	"bisonai.com/orakl/node/pkg/wfetcher/providers/binance"
	"bisonai.com/orakl/node/pkg/wfetcher/providers/coinbase"
	"bisonai.com/orakl/node/pkg/wfetcher/providers/coinone"
	"bisonai.com/orakl/node/pkg/wfetcher/providers/korbit"
	"github.com/rs/zerolog/log"
)

const (
	StoreInterval = 500 * time.Millisecond
)

type App struct {
	fetchers []common.FetcherInterface
	buffer   chan common.FeedData
}

func New() *App {
	return &App{
		buffer: make(chan common.FeedData, common.BufferSize),
	}
}

func (a *App) Init(ctx context.Context) error {
	// TODO: Proxy support
	feeds, err := db.QueryRows[common.Feed](ctx, common.GetAllFeedsQuery, nil)
	if err != nil {
		log.Error().Err(err).Msg("error in fetching feeds")
		return err
	}
	feedMap := common.GetWssFeedMap(feeds)

	fetcherCreators := map[string]func(context.Context, ...common.FetcherOption) (common.FetcherInterface, error){
		"binance":  binance.New,
		"coinbase": coinbase.New,
		"coinone":  coinone.New,
		"korbit":   korbit.New,
	}

	for name, creator := range fetcherCreators {
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

	ticker := time.NewTicker(StoreInterval)
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
	for {
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
}
