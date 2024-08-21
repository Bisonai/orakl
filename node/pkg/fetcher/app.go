package fetcher

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/bus"
	"bisonai.com/miko/node/pkg/db"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/websocketfetcher"
	"github.com/rs/zerolog/log"
)

func New(bus *bus.MessageBus) *App {
	return &App{
		Fetchers:         make(map[int32]*Fetcher, 0),
		WebsocketFetcher: websocketfetcher.New(),
		LatestFeedDataMap: &LatestFeedDataMap{
			FeedDataMap: make(map[int32]*FeedData),
			Mu:          sync.RWMutex{},
		},
		FeedDataDumpChannel: make(chan *FeedData, DefaultFeedDataDumpChannelSize),
		Bus:                 bus,
	}
}

func (a *App) Run(ctx context.Context) error {
	err := a.initialize(ctx)
	if err != nil {
		return err
	}

	a.subscribe(ctx)

	return a.startAll(ctx)
}

func (a *App) subscribe(ctx context.Context) {
	log.Debug().Str("Player", "Fetcher").Msg("fetcher subscribing to message bus")
	channel := a.Bus.Subscribe(bus.FETCHER)
	go func() {
		log.Debug().Str("Player", "Fetcher").Msg("fetcher message bus subscription goroutine started")
		for {
			select {
			case msg := <-channel:
				log.Debug().
					Str("Player", "Fetcher").
					Str("from", msg.From).
					Str("to", msg.To).
					Str("command", msg.Content.Command).
					Msg("fetcher received message")
				go a.handleMessage(ctx, msg)
			case <-ctx.Done():
				log.Debug().Str("Player", "Fetcher").Msg("fetcher message bus subscription goroutine stopped")
				return
			}
		}
	}()
}

func (a *App) handleMessage(ctx context.Context, msg bus.Message) {
	if msg.From != bus.ADMIN {
		log.Debug().Str("Player", "Fetcher").Msg("fetcher received message from non-admin")
		return
	}

	if msg.To != bus.FETCHER {
		log.Debug().Str("Player", "Fetcher").Msg("message not for fetcher")
		return
	}

	switch msg.Content.Command {
	case bus.ACTIVATE_FETCHER:
		log.Debug().Str("Player", "Fetcher").Msg("activate fetcher msg received")
		configId, err := bus.ParseInt32MsgParam(msg, "id")
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to parse configId")
			bus.HandleMessageError(err, msg, "failed to parse configId")
			return
		}

		log.Debug().Str("Player", "Fetcher").Int32("configId", configId).Msg("activating fetcher")
		err = a.startFetcherById(ctx, configId)
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to start fetcher")
			bus.HandleMessageError(err, msg, "failed to start fetcher")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.DEACTIVATE_FETCHER:
		log.Debug().Str("Player", "Fetcher").Msg("deactivate fetcher msg received")
		configId, err := bus.ParseInt32MsgParam(msg, "id")
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to parse configId")
			bus.HandleMessageError(err, msg, "failed to parse configId")
			return
		}

		log.Debug().Str("Player", "Fetcher").Int32("configId", configId).Msg("deactivating fetcher")
		err = a.stopFetcherById(ctx, configId)
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to stop fetcher")
			bus.HandleMessageError(err, msg, "failed to stop fetcher")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.STOP_FETCHER_APP:
		log.Debug().Str("Player", "Fetcher").Msg("stopping all fetchers")
		err := a.stopAll(ctx)
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to stop all fetchers")
			bus.HandleMessageError(err, msg, "failed to stop all fetchers")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.START_FETCHER_APP:
		log.Debug().Str("Player", "Fetcher").Msg("starting all fetchers")
		err := a.startAll(ctx)
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to start all fetchers")
			bus.HandleMessageError(err, msg, "failed to start all fetchers")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.REFRESH_FETCHER_APP:
		err := a.stopAll(ctx)
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to stop all fetchers")
			bus.HandleMessageError(err, msg, "failed to stop all fetchers")
			return
		}
		err = a.initialize(ctx)
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to initialize fetchers")
			bus.HandleMessageError(err, msg, "failed to initialize fetchers")
			return
		}
		err = a.startAll(ctx)
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to start all fetchers")
			bus.HandleMessageError(err, msg, "failed to start all fetchers")
			return
		}

		log.Debug().Str("Player", "Fetcher").Msg("refreshing fetcher")
		msg.Response <- bus.MessageResponse{Success: true}
	}
}

func (a *App) startAll(ctx context.Context) error {
	err := a.startAllFetchers(ctx)
	if err != nil {
		return err
	}

	a.startLocalAggregateBulkWriter(ctx)

	err = a.startAllLocalAggregators(ctx)
	if err != nil {
		return err
	}

	go a.WebsocketFetcher.Start(ctx)

	return a.startFeedDataBulkWriter(ctx)
}

func (a *App) stopAll(ctx context.Context) error {
	err := a.stopAllFetchers(ctx)
	if err != nil {
		return err
	}

	err = a.stopAllLocalAggregators(ctx)
	if err != nil {
		return err
	}

	err = a.stopLocalAggregateBulkWriter()
	if err != nil {
		return err
	}

	return a.stopFeedDataBulkWriter(ctx)
}

func (a *App) startFetcher(ctx context.Context, fetcher *Fetcher) error {
	if fetcher.isRunning {
		log.Debug().Str("Player", "Fetcher").Str("fetcher", fetcher.Name).Msg("fetcher already running")
		return nil
	}

	fetcher.Run(ctx, a.Proxies)

	log.Debug().Str("Player", "Fetcher").Str("fetcher", fetcher.Name).Msg("fetcher started")
	return nil
}

func (a *App) startLocalAggregator(ctx context.Context, localAggregator *LocalAggregator) error {
	if localAggregator.isRunning {
		log.Debug().Str("Player", "Fetcher").Str("localAggregator", localAggregator.Name).Msg("localAggregator already running")
		return nil
	}

	localAggregator.Run(ctx)

	log.Debug().Str("Player", "Fetcher").Str("localAggregator", localAggregator.Name).Msg("localAggregator started")
	return nil
}

func (a *App) startFeedDataBulkWriter(ctx context.Context) error {
	if a.FeedDataBulkWriter.isRunning {
		log.Debug().Str("Player", "Fetcher").Msg("feedDataBulkWriter already running")
		return nil
	}

	a.FeedDataBulkWriter.Run(ctx)

	log.Debug().Str("Player", "Fetcher").Msg("feedDataBulkWriter started")
	return nil
}

func (a *App) startLocalAggregateBulkWriter(ctx context.Context) {
	if a.LocalAggregateBulkWriter.isRunning {
		log.Debug().Str("Player", "Fetcher").Msg("LocalAggregateBulkWriter already running")
	}

	go a.LocalAggregateBulkWriter.Run(ctx)

	log.Debug().Str("Player", "Fetcher").Msg("LocalAggregateBulkWriter started")
}

func (a *App) startFetcherById(ctx context.Context, configId int32) error {
	if fetcher, ok := a.Fetchers[configId]; ok {
		return a.startFetcher(ctx, fetcher)
	}
	log.Error().Str("Player", "Fetcher").Int32("adapterId", configId).Msg("fetcher not found")
	return errorSentinel.ErrFetcherNotFound
}

func (a *App) startAllFetchers(ctx context.Context) error {
	for _, fetcher := range a.Fetchers {
		err := a.startFetcher(ctx, fetcher)
		if err != nil {
			log.Error().Str("Player", "Fetcher").Err(err).Str("fetcher", fetcher.Name).Msg("failed to start fetcher")
			return err
		}
		// starts with random sleep to avoid all fetchers starting at the same time
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)+50))
	}
	return nil
}

func (a *App) startAllLocalAggregators(ctx context.Context) error {
	for _, localAggregator := range a.LocalAggregators {
		err := a.startLocalAggregator(ctx, localAggregator)
		if err != nil {
			log.Error().Str("Player", "Fetcher").Err(err).Str("localAggregator", localAggregator.Name).Msg("failed to start localAggregator")
			return err
		}
		// starts with random sleep to avoid all fetchers starting at the same time
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)+50))
	}
	return nil
}

func (a *App) stopFetcher(ctx context.Context, fetcher *Fetcher) error {
	log.Debug().Str("fetcher", fetcher.Name).Msg("stopping fetcher")
	if !fetcher.isRunning {
		log.Debug().Str("Player", "Fetcher").Str("fetcher", fetcher.Name).Msg("fetcher already stopped")
		return nil
	}
	if fetcher.cancel == nil {
		return errorSentinel.ErrFetcherCancelNotFound
	}
	fetcher.cancel()
	fetcher.isRunning = false
	return nil
}

func (a *App) stopLocalAggregator(ctx context.Context, localAggregator *LocalAggregator) error {
	log.Debug().Str("LocalAggregator", localAggregator.Name).Msg("stopping LocalAggregator")
	if !localAggregator.isRunning {
		log.Debug().Str("Player", "Fetcher").Str("localAggregator", localAggregator.Name).Msg("localAggregator already stopped")
		return nil
	}
	if localAggregator.cancel == nil {
		return errorSentinel.ErrLocalAggregatorCancelNotFound
	}
	localAggregator.cancel()
	localAggregator.isRunning = false
	return nil
}

func (a *App) stopFeedDataBulkWriter(ctx context.Context) error {
	log.Debug().Msg("stopping feed data bulk writer")
	if !a.FeedDataBulkWriter.isRunning {
		log.Debug().Str("Player", "Fetcher").Msg("feed data bulk writer already stopped")
		return nil
	}
	if a.FeedDataBulkWriter.cancel == nil {
		return errorSentinel.ErrFeedDataBulkWriterCancelNotFound
	}
	a.FeedDataBulkWriter.cancel()
	a.FeedDataBulkWriter.isRunning = false
	return nil
}

func (a *App) stopLocalAggregateBulkWriter() error {
	log.Debug().Msg("stopping LocalAggregateBulkWriter")
	if !a.LocalAggregateBulkWriter.isRunning {
		log.Debug().Str("Player", "Fetcher").Msg("LocalAggregateBulkWriter already stopped")
		return nil
	}
	if a.LocalAggregateBulkWriter.cancel == nil {
		return errorSentinel.ErrLocalAggregateBulkWriterCancelNotFound
	}
	a.LocalAggregateBulkWriter.cancel()
	a.LocalAggregateBulkWriter.isRunning = false
	return nil
}

func (a *App) stopFetcherById(ctx context.Context, configId int32) error {
	if fetcher, ok := a.Fetchers[configId]; ok {
		return a.stopFetcher(ctx, fetcher)
	}
	return errorSentinel.ErrFetcherNotFound
}

func (a *App) stopAllFetchers(ctx context.Context) error {
	for _, fetcher := range a.Fetchers {
		err := a.stopFetcher(ctx, fetcher)
		if err != nil {
			log.Error().Str("Player", "Fetcher").Err(err).Str("fetcher", fetcher.Name).Msg("failed to stop fetcher")
			return err
		}
	}
	return nil
}

func (a *App) stopAllLocalAggregators(ctx context.Context) error {
	for _, localAggregator := range a.LocalAggregators {
		err := a.stopLocalAggregator(ctx, localAggregator)
		if err != nil {
			log.Error().Str("Player", "Fetcher").Err(err).Str("localAggregator", localAggregator.Name).Msg("failed to stop localAggregator")
			return err
		}
	}
	return nil
}

func (a *App) getConfigs(ctx context.Context) ([]Config, error) {
	configs, err := db.QueryRows[Config](ctx, SelectConfigsQuery, nil)
	if err != nil {
		return nil, err
	}
	return configs, err
}

func (a *App) getFeedsWithoutWss(ctx context.Context, configId int32) ([]Feed, error) {
	feeds, err := db.QueryRows[Feed](ctx, SelectHttpRequestFeedsByConfigIdQuery, map[string]any{"config_id": configId})
	if err != nil {
		return nil, err
	}

	return feeds, err
}

func (a *App) getFeeds(ctx context.Context, configId int32) ([]Feed, error) {
	feeds, err := db.QueryRows[Feed](ctx, SelectFeedsByConfigIdQuery, map[string]any{"config_id": configId})
	if err != nil {
		return nil, err
	}
	return feeds, err
}

func (a *App) getProxies(ctx context.Context) ([]Proxy, error) {
	proxies, err := db.QueryRows[Proxy](ctx, SelectAllProxiesQuery, nil)
	if err != nil {
		return nil, err
	}
	return proxies, err
}

func (a *App) initialize(ctx context.Context) error {
	configs, err := a.getConfigs(ctx)
	if err != nil {
		return err
	}

	a.Fetchers = make(map[int32]*Fetcher, len(configs))
	a.LocalAggregators = make(map[int32]*LocalAggregator, len(configs))
	a.LocalAggregateBulkWriter = NewLocalAggregateBulkWriter(DefaultLocalAggregateInterval)
	a.LocalAggregateBulkWriter.localAggregatesChannel = make(chan *LocalAggregate, LocalAggregatesChannelSize)

	for _, config := range configs {
		// for fetcher it'll get fetcherFeeds without websocket fetcherFeeds
		fetcherFeeds, getFeedsErr := a.getFeedsWithoutWss(ctx, config.ID)
		if getFeedsErr != nil {
			return getFeedsErr
		}

		if len(fetcherFeeds) > 0 {
			a.Fetchers[config.ID] = NewFetcher(config, fetcherFeeds, a.LatestFeedDataMap, a.FeedDataDumpChannel)
		}

		// for localAggregator it'll get all feeds to be collected
		localAggregatorFeeds, getFeedsErr := a.getFeeds(ctx, config.ID)
		if getFeedsErr != nil {
			return getFeedsErr
		}
		a.LocalAggregators[config.ID] = NewLocalAggregator(config, localAggregatorFeeds, a.LocalAggregateBulkWriter.localAggregatesChannel, a.Bus, a.LatestFeedDataMap)
	}
	feedDataDumpIntervalRaw := os.Getenv("FEED_DATA_STREAM_INTERVAL")
	dumpInterval, err := time.ParseDuration(feedDataDumpIntervalRaw)
	if err != nil {
		dumpInterval = DefaultFeedDataDumpInterval
	}
	a.FeedDataBulkWriter = NewFeedDataBulkWriter(dumpInterval, a.FeedDataDumpChannel)

	proxies, getProxyErr := a.getProxies(ctx)
	if getProxyErr != nil {
		return getProxyErr
	}
	a.Proxies = proxies

	err = a.WebsocketFetcher.Init(ctx, websocketfetcher.WithLatestFeedDataMap(a.LatestFeedDataMap), websocketfetcher.WithFeedDataDumpChannel(a.FeedDataDumpChannel))
	if err != nil {
		return err
	}

	return nil
}

func (a *App) String() string {
	return fmt.Sprintf("%+v\n", a.Fetchers)
}
