package fetcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/utils"
	"github.com/rs/zerolog/log"
)

const FETCHER_FREQUENCY = 2 * time.Second

func New(bus *bus.MessageBus) *Fetcher {
	return &Fetcher{
		Adapters: make([]AdapterDetail, 0),
		Bus:      bus,
	}
}

func (f *Fetcher) Run(ctx context.Context) error {
	err := f.initialize(ctx)
	if err != nil {
		return err
	}

	for _, adapter := range f.Adapters {
		go f.runAdapter(ctx, adapter)
		// 100 ~ 400 ms delay between launching each adapters
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(300)+100))
	}

	return nil
}

func (f *Fetcher) runAdapter(ctx context.Context, adapter AdapterDetail) {
	ticker := time.NewTicker(FETCHER_FREQUENCY)
	defer ticker.Stop()

	for range ticker.C {
		err := f.fetchAndInsert(ctx, adapter)
		if err != nil {
			log.Error().Err(err).Msg("failed to fetch and insert")
		}
	}
}

func (f *Fetcher) fetchAndInsert(ctx context.Context, adapter AdapterDetail) error {
	results, err := f.fetch(adapter)
	if err != nil {
		return err
	}
	aggregated, err := utils.GetFloatAvg(results)
	if err != nil {
		return err
	}
	err = f.insertPgsql(ctx, adapter.Name, aggregated)
	if err != nil {
		return err
	}
	err = f.insertRdb(ctx, adapter.Name, aggregated)
	if err != nil {
		return err
	}
	return nil
}

func (f *Fetcher) insertPgsql(ctx context.Context, name string, value float64) error {
	err := db.QueryWithoutResult(ctx, InsertLocalAggregateQuery, map[string]any{"name": name, "value": int64(value)})
	return err
}

func (f *Fetcher) insertRdb(ctx context.Context, name string, value float64) error {
	key := "latestAggregate:" + name
	data, err := json.Marshal(redisAggregate{Value: int64(value), Timestamp: time.Now()})
	if err != nil {
		return err
	}
	return db.Set(ctx, key, string(data), time.Duration(5*time.Minute))
}

func (f *Fetcher) fetch(adapter AdapterDetail) ([]float64, error) {
	adapterFeeds := adapter.Feeds

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

func (f *Fetcher) getAdapters(ctx context.Context) ([]Adapter, error) {
	adapters, err := db.QueryRows[Adapter](ctx, SelectActiveAdaptersQuery, nil)
	if err != nil {
		return nil, err
	}
	return adapters, err
}

func (f *Fetcher) getFeeds(ctx context.Context, adapterId int64) ([]Feed, error) {
	feeds, err := db.QueryRows[Feed](ctx, SelectFeedsByAdapterIdQuery, map[string]any{"adapterId": adapterId})
	if err != nil {
		return nil, err
	}

	return feeds, err
}

func (f *Fetcher) initialize(ctx context.Context) error {
	adapters, err := f.getAdapters(ctx)
	if err != nil {
		return err
	}
	f.Adapters = make([]AdapterDetail, 0, len(adapters))
	for _, adapter := range adapters {
		feeds, err := f.getFeeds(ctx, adapter.ID)
		if err != nil {
			return err
		}
		f.Adapters = append(f.Adapters, AdapterDetail{adapter, feeds})
	}
	return nil
}

func (f *Fetcher) String() string {
	return fmt.Sprintf("%+v\n", f.Adapters)
}
