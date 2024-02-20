package fetcher

import (
	"context"
	"encoding/json"
	"fmt"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/utils"
)

const (
	loadActiveAdaptersQuery   = `SELECT * FROM adapters WHERE active = true`
	loadFeedsByAdapterIdQuery = `SELECT * FROM feeds WHERE adapter_id = @adapterId`
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

func (f *Fetcher) Run() error {
	f.initialize(context.Background())
	return nil
}

func (f *Fetcher) Stop() error {
	return nil
}

func (f *Fetcher) StartAdapter(adapterName string) error {
	for _, adapter := range f.Adapters {
		f.fetch(adapter)
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

		result, err := f.reduceAll(res, definition.Reducers)
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

func (f *Fetcher) StopAdapter(adapterName string) error {
	return nil
}

func (f *Fetcher) Refresh() error {
	return nil
}

func (f *Fetcher) sendDeviationSignal() error {
	return nil
}

func (f *Fetcher) loadActiveAdapters() error {
	return nil
}

func (f *Fetcher) getAdapters(ctx context.Context) ([]Adapter, error) {
	adapters, err := db.QueryRows[Adapter](ctx, loadActiveAdaptersQuery, nil)
	if err != nil {
		return nil, err
	}
	return adapters, err
}

func (f *Fetcher) getFeeds(ctx context.Context, adapterId int64) ([]Feed, error) {
	feeds, err := db.QueryRows[Feed](ctx, loadFeedsByAdapterIdQuery, map[string]any{"id": adapterId})
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
