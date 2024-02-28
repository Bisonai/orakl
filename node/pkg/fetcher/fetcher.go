package fetcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/utils"
	"github.com/rs/zerolog/log"
)

const FETCHER_FREQUENCY = 2 * time.Second

func New(bus *bus.MessageBus) *App {
	return &App{
		Fetchers: make(map[int64]*Fetcher, 0),
		Bus:      bus,
	}
}

func (f *App) Run(ctx context.Context) error {
	err := f.initialize(ctx)
	if err != nil {
		return err
	}

	f.subscribe(ctx)

	for _, adapter := range f.Fetchers {
		err = f.startFetcher(ctx, adapter)
		if err != nil {
			log.Error().Err(err).Str("name", adapter.Name).Msg("failed to start adapter")
		}
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(300)+100))
	}

	return nil
}

func (f *App) subscribe(ctx context.Context) {
	log.Debug().Msg("fetcher subscribing to message bus")
	channel := f.Bus.Subscribe(bus.FETCHER)
	go func() {
		log.Debug().Msg("fetcher message bus subscription goroutine started")
		for {
			select {
			case msg := <-channel:
				log.Debug().
					Str("from", msg.From).
					Str("to", msg.To).
					Str("command", msg.Content.Command).
					Msg("fetcher received message")
				f.handleMessage(ctx, msg)
			case <-ctx.Done():
				log.Debug().Msg("fetcher message bus subscription goroutine stopped")
				return
			}
		}
	}()
}

func (f *App) handleMessage(ctx context.Context, msg bus.Message) {
	if msg.From != bus.ADMIN {
		log.Debug().Msg("fetcher received message from non-admin")
		return
	}

	if msg.To != bus.FETCHER {
		log.Debug().Msg("message not for fetcher")
		return
	}

	switch msg.Content.Command {
	case bus.ACTIVATE_FETCHER:
		log.Debug().Msg("activate adapter msg received")
		adapterId, err := f.parseIdMsgParam(msg)
		if err != nil {
			log.Error().Err(err).Msg("failed to parse adapterId")
			return
		}

		log.Debug().Int64("adapterId", adapterId).Msg("activating adapter")
		err = f.startFetcherById(ctx, adapterId)
		if err != nil {
			log.Error().Err(err).Msg("failed to start adapter")
		}
	case bus.DEACTIVATE_FETCHER:
		log.Debug().Msg("deactivate adapter msg received")
		adapterId, err := f.parseIdMsgParam(msg)
		if err != nil {
			log.Error().Err(err).Msg("failed to parse adapterId")
			return
		}

		log.Debug().Int64("adapterId", adapterId).Msg("deactivating adapter")
		err = f.stopFetcherById(ctx, adapterId)
		if err != nil {
			log.Error().Err(err).Msg("failed to stop adapter")
		}
	case bus.STOP_FETCHER_APP:
		// TODO: stop fetcher

		log.Debug().Msg("stopping fetcher")
	case bus.START_FETCHER_APP:
		// TODO: start fetcher

		log.Debug().Msg("starting fetcher")
	case bus.REFRESH_FETCHER_APP:
		// TODO: refresh adapters

		log.Debug().Msg("refreshing fetcher")
	}
}

func (f *App) startFetcher(ctx context.Context, fetcher *Fetcher) error {
	if fetcher.isRunning {
		log.Debug().Str("adapter", fetcher.Name).Msg("adapter already running")
		return errors.New("adapter already running")
	}
	adapterCtx, cancel := context.WithCancel(ctx)
	fetcher.adapterCtx = adapterCtx
	fetcher.cancel = cancel
	fetcher.isRunning = true

	ticker := time.NewTicker(FETCHER_FREQUENCY)

	go func() {
		log.Debug().Str("adapter", fetcher.Name).Msg("starting adapter goroutine")
		for {
			select {
			case <-ticker.C:
				log.Debug().Str("adapter", fetcher.Name).Msg("fetching and inserting")
				err := f.fetchAndInsert(adapterCtx, *fetcher)
				if err != nil {
					log.Error().Err(err).Msg("failed to fetch and insert")
				}
			case <-adapterCtx.Done():
				log.Debug().Str("adapter", fetcher.Name).Msg("adapter stopped")
				return
			}
		}
	}()

	log.Debug().Str("adapter", fetcher.Name).Msg("adapter started")
	return nil
}

func (f *App) startFetcherById(ctx context.Context, adapterId int64) error {
	if adapter, ok := f.Fetchers[adapterId]; ok {
		return f.startFetcher(ctx, adapter)
	}
	return errors.New("adapter not found")
}

func (f *App) stopFetcher(ctx context.Context, fetcher *Fetcher) error {
	log.Debug().Str("adapter", fetcher.Name).Msg("stopping adapter")
	if !fetcher.isRunning {
		return errors.New("adapter already stopped")
	}
	if fetcher.cancel == nil {
		return errors.New("adapter cancel function not found")
	}
	fetcher.cancel()
	fetcher.isRunning = false
	return nil
}

func (f *App) stopFetcherById(ctx context.Context, adapterId int64) error {
	if adapter, ok := f.Fetchers[adapterId]; ok {
		return f.stopFetcher(ctx, adapter)
	}
	return errors.New("adapter not found")
}

func (f *App) fetchAndInsert(ctx context.Context, fetcher Fetcher) error {
	log.Debug().Str("adapter", fetcher.Name).Msg("fetching and inserting")

	results, err := f.fetch(fetcher)
	if err != nil {
		return err
	}
	log.Debug().Str("adapter", fetcher.Name).Msg("fetch complete")

	aggregated, err := utils.GetFloatAvg(results)
	if err != nil {
		return err
	}
	log.Debug().Str("adapter", fetcher.Name).Float64("aggregated", aggregated).Msg("aggregated")

	err = f.insertPgsql(ctx, fetcher.Name, aggregated)
	if err != nil {
		return err
	}
	log.Debug().Str("adapter", fetcher.Name).Msg("inserted into pgsql")

	err = f.insertRdb(ctx, fetcher.Name, aggregated)
	if err != nil {
		return err
	}
	log.Debug().Str("adapter", fetcher.Name).Msg("inserted into rdb")
	return nil
}

func (f *App) insertPgsql(ctx context.Context, name string, value float64) error {
	err := db.QueryWithoutResult(ctx, InsertLocalAggregateQuery, map[string]any{"name": name, "value": int64(value)})
	return err
}

func (f *App) insertRdb(ctx context.Context, name string, value float64) error {
	key := "latestAggregate:" + name
	data, err := json.Marshal(redisAggregate{Value: int64(value), Timestamp: time.Now()})
	if err != nil {
		return err
	}
	return db.Set(ctx, key, string(data), time.Duration(5*time.Minute))
}

func (f *App) fetch(fetcher Fetcher) ([]float64, error) {
	adapterFeeds := fetcher.Feeds

	data := []float64{}

	for _, feed := range adapterFeeds {
		definition := new(Definition)
		err := json.Unmarshal(feed.Definition, &definition)
		if err != nil {
			log.Error().Err(err).Msg("failed to unmarshal feed definition")
			continue
		}
		res, err := utils.GetRequest[interface{}](definition.Url, nil, definition.Headers)
		if err != nil {
			log.Error().Err(err).Msg("failed to get request")
			continue
		}

		result, err := utils.Reduce(res, definition.Reducers)
		if err != nil {
			log.Error().Err(err).Msg("failed to reduce")
			continue
		}

		data = append(data, result)
	}
	if len(data) < 1 {
		return nil, errors.New("no data fetched")
	}
	return data, nil
}

func (f *App) getAdapters(ctx context.Context) ([]Adapter, error) {
	adapters, err := db.QueryRows[Adapter](ctx, SelectActiveAdaptersQuery, nil)
	if err != nil {
		return nil, err
	}
	return adapters, err
}

func (f *App) getFeeds(ctx context.Context, adapterId int64) ([]Feed, error) {
	feeds, err := db.QueryRows[Feed](ctx, SelectFeedsByAdapterIdQuery, map[string]any{"adapterId": adapterId})
	if err != nil {
		return nil, err
	}

	return feeds, err
}

func (f *App) initialize(ctx context.Context) error {
	adapters, err := f.getAdapters(ctx)
	if err != nil {
		return err
	}
	f.Fetchers = make(map[int64]*Fetcher, len(adapters))
	for _, adapter := range adapters {
		feeds, err := f.getFeeds(ctx, adapter.ID)
		if err != nil {
			return err
		}

		f.Fetchers[adapter.ID] = &Fetcher{
			Adapter:   adapter,
			Feeds:     feeds,
			isRunning: false,
		}
	}
	return nil
}

func (f *App) String() string {
	return fmt.Sprintf("%+v\n", f.Fetchers)
}

func (f *App) parseIdMsgParam(msg bus.Message) (int64, error) {
	rawId, ok := msg.Content.Args["id"]
	if !ok {
		return 0, errors.New("adapterId not found in message")
	}

	adapterIdPayload, ok := rawId.(string)
	if !ok {
		return 0, errors.New("failed to convert adapter id to string")
	}

	adapterId, err := strconv.ParseInt(adapterIdPayload, 10, 64)
	if err != nil {
		return 0, errors.New("failed to parse adapterId")
	}

	return adapterId, nil
}
