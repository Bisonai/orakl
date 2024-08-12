package fetcher

import (
	"context"
	"fmt"

	"math/rand"
	"os"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/websocketfetcher"
	"github.com/rs/zerolog/log"
)

const LocalAggregatesChannelSize = 2_000
const DefaultLocalAggregateInterval = 200 * time.Millisecond

func New(bus *bus.MessageBus) *App {
	return &App{
		Fetchers:         make(map[int32]*Fetcher, 0),
		WebsocketFetcher: websocketfetcher.New(),
		Bus:              bus,
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

	err = a.startAllCollectors(ctx)
	if err != nil {
		return err
	}

	go a.WebsocketFetcher.Start(ctx)

	return a.startStreamer(ctx)
}

func (a *App) stopAll(ctx context.Context) error {
	err := a.stopAllFetchers(ctx)
	if err != nil {
		return err
	}

	err = a.stopAllCollectors(ctx)
	if err != nil {
		return err
	}

	err = a.stopLocalAggregateBulkWriter()
	if err != nil {
		return err
	}

	return a.stopStreamer(ctx)
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

func (a *App) startCollector(ctx context.Context, collector *Collector) error {
	if collector.isRunning {
		log.Debug().Str("Player", "Collector").Str("collector", collector.Name).Msg("collector already running")
		return nil
	}

	collector.Run(ctx)

	log.Debug().Str("Player", "Collector").Str("collector", collector.Name).Msg("collector started")
	return nil
}

func (a *App) startStreamer(ctx context.Context) error {
	if a.Streamer.isRunning {
		log.Debug().Str("Player", "Streamer").Msg("streamer already running")
		return nil
	}

	a.Streamer.Run(ctx)

	log.Debug().Str("Player", "Streamer").Msg("streamer started")
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

func (a *App) startAllCollectors(ctx context.Context) error {
	for _, collector := range a.Collectors {
		err := a.startCollector(ctx, collector)
		if err != nil {
			log.Error().Str("Player", "Collector").Err(err).Str("collector", collector.Name).Msg("failed to start collector")
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

func (a *App) stopCollector(ctx context.Context, collector *Collector) error {
	log.Debug().Str("collector", collector.Name).Msg("stopping collector")
	if !collector.isRunning {
		log.Debug().Str("Player", "Collector").Str("collector", collector.Name).Msg("collector already stopped")
		return nil
	}
	if collector.cancel == nil {
		return errorSentinel.ErrCollectorCancelNotFound
	}
	collector.cancel()
	collector.isRunning = false
	return nil
}

func (a *App) stopStreamer(ctx context.Context) error {
	log.Debug().Msg("stopping streamer")
	if !a.Streamer.isRunning {
		log.Debug().Str("Player", "Streamer").Msg("streamer already stopped")
		return nil
	}
	if a.Streamer.cancel == nil {
		return errorSentinel.ErrStreamerCancelNotFound
	}
	a.Streamer.cancel()
	a.Streamer.isRunning = false
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

func (a *App) stopAllCollectors(ctx context.Context) error {
	for _, collector := range a.Collectors {
		err := a.stopCollector(ctx, collector)
		if err != nil {
			log.Error().Str("Player", "Collector").Err(err).Str("collector", collector.Name).Msg("failed to stop collector")
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
	a.Collectors = make(map[int32]*Collector, len(configs))
	a.LocalAggregateBulkWriter = NewLocalAggregateBulkWriter(DefaultLocalAggregateInterval)
	a.LocalAggregateBulkWriter.localAggregatesChannel = make(chan *LocalAggregate, LocalAggregatesChannelSize)

	for _, config := range configs {
		// for fetcher it'll get fetcherFeeds without websocket fetcherFeeds
		fetcherFeeds, getFeedsErr := a.getFeedsWithoutWss(ctx, config.ID)
		if getFeedsErr != nil {
			return getFeedsErr
		}

		if len(fetcherFeeds) > 0 {
			a.Fetchers[config.ID] = NewFetcher(config, fetcherFeeds)
		}

		// for collector it'll get all feeds to be collected
		collectorFeeds, getFeedsErr := a.getFeeds(ctx, config.ID)
		if getFeedsErr != nil {
			return getFeedsErr
		}
		a.Collectors[config.ID] = NewCollector(config, collectorFeeds, a.LocalAggregateBulkWriter.localAggregatesChannel, a.Bus)
	}
	streamIntervalRaw := os.Getenv("FEED_DATA_STREAM_INTERVAL")
	streamInterval, err := time.ParseDuration(streamIntervalRaw)
	if err != nil {
		streamInterval = DefaultStreamInterval
	}
	a.Streamer = NewStreamer(streamInterval)

	proxies, getProxyErr := a.getProxies(ctx)
	if getProxyErr != nil {
		return getProxyErr
	}
	a.Proxies = proxies

	err = a.WebsocketFetcher.Init(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (a *App) String() string {
	return fmt.Sprintf("%+v\n", a.Fetchers)
}
