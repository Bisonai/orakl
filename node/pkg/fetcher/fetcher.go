package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/utils"
)

const (
	SelectActiveAdaptersQuery   = `SELECT * FROM adapters WHERE active = true`
	SelectFeedsByAdapterIdQuery = `SELECT * FROM feeds WHERE adapter_id = @adapterId`
	InsertLocalAggregateQuery   = `INSERT INTO local_aggregates (name, value) VALUES (@name, @value)`
)

type Adapter struct {
	ID     int64  `db:"id"`
	Name   string `db:"name"`
	Active bool   `db:"active"`
}

type AdapterDetail struct {
	Adapter
	Feeds []Feed
}

type Feed struct {
	ID         int64           `db:"id"`
	Name       string          `db:"name"`
	Definition json.RawMessage `db:"definition"`
	AdapterID  int64           `db:"adapter_id"`
}

type Fetcher struct {
	Bus      *bus.MessageBus
	Adapters []AdapterDetail
}

func NewFetcher(bus *bus.MessageBus) *Fetcher {
	return &Fetcher{
		Adapters: make([]AdapterDetail, 0),
		Bus:      bus,
	}
}

func (f *Fetcher) Run(ctx context.Context) {
	f.initialize(ctx)
	ticker := time.NewTicker(2 * time.Second)

	go func() {
		for range ticker.C {
			err := f.runAdapter(ctx)
			if err != nil {
				fmt.Println(err)
			}
		}
	}()
}

func (f *Fetcher) runAdapter(ctx context.Context) error {
	for _, adapter := range f.Adapters {
		result, err := f.fetch(adapter)
		if err != nil {
			return err
		}
		aggregated := getAvg(result)
		_, err = db.Query(ctx, InsertLocalAggregateQuery, map[string]any{"name": adapter.Name, "value": int64(aggregated)})
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *Fetcher) fetch(adapter AdapterDetail) ([]float64, error) {
	adapterFeeds := adapter.Feeds

	data := []float64{}

	for _, feed := range adapterFeeds {
		definition := new(Definition)
		err := json.Unmarshal(feed.Definition, &definition)
		if err != nil {
			fmt.Println(err)
			continue
		}
		res, err := utils.GetRequest[interface{}](definition.Url, nil, definition.Headers)
		if err != nil {
			fmt.Println(err)
			continue
		}

		result, err := ReduceAll(res, definition.Reducers)
		if err != nil {
			fmt.Println(err)
			continue
		}

		data = append(data, result)
	}
	if len(data) < 1 {
		return nil, fmt.Errorf("no data fetched")
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
	feeds, err := db.QueryRows[Feed](ctx, SelectFeedsByAdapterIdQuery, map[string]any{"id": adapterId})
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
