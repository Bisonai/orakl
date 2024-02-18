package fetcher

import (
	"context"
	"encoding/json"
	"fmt"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
)

const (
	GetAdapters         = `GET * FROM adapters;`
	GetFeedsByAdapterId = `GET * FROM feeds WHERE adapter_id = @id;`
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
	return nil
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
	adapters, err := db.QueryRows[Adapter](ctx, GetAdapters, nil)
	if err != nil {
		return nil, err
	}
	return adapters, err
}

func (f *Fetcher) getFeeds(ctx context.Context, adapterId int64) ([]Feed, error) {
	feeds, err := db.QueryRows[Feed](ctx, GetFeedsByAdapterId, map[string]any{"id": adapterId})
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
