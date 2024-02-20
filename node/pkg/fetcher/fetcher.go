package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/utils"
)

const (
	loadActiveAdaptersQuery   = `SELECT * FROM adapters WHERE active = true`
	loadFeedsByAdapterIdQuery = `SELECT * FROM feeds WHERE adapter_id = @adapterId`
)

type Definition struct {
	Url      string            `json:"url"`
	Headers  map[string]string `json:"headers"`
	Method   string            `json:"method"`
	Reducers []Reducer         `json:"reducers"`
}

type Reducer struct {
	Function string      `json:"function"`
	Args     interface{} `json:"args"`
}

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

func (f *Fetcher) reduceAll(raw interface{}, reducers []Reducer) (float64, error) {
	var result float64
	for _, reducer := range reducers {
		var err error
		raw, err = f.reduce(raw, reducer)
		if err != nil {
			return 0, err
		}
	}
	result, ok := raw.(float64)
	if !ok {
		return 0, fmt.Errorf("cannot cast raw data to float")
	}
	return result, nil
}

func (f *Fetcher) reduce(raw interface{}, reducer Reducer) (interface{}, error) {
	switch reducer.Function {
	case "INDEX":
		castedRaw, ok := raw.([]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot cast raw data to []interface{}")
		}
		return castedRaw[reducer.Args.(int)], nil
	case "PARSE", "PATH":
		castedRaw, ok := raw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot cast raw data to map[string]interface{}")
		}
		args := reducer.Args.([]string)
		for _, arg := range args {
			castedRaw = castedRaw[arg].(map[string]interface{})
		}
		return castedRaw, nil
	case "MUL":
		castedRaw, ok := raw.(float64)
		if !ok {
			return nil, fmt.Errorf("cannot cast raw data to float")
		}
		return castedRaw * reducer.Args.(float64), nil
	case "POW10":
		castedRaw, ok := raw.(float64)
		if !ok {
			return nil, fmt.Errorf("cannot cast raw data to float")
		}
		return float64(math.Pow10(reducer.Args.(int))) * castedRaw, nil
	case "ROUND":
		castedRaw, ok := raw.(float64)
		if !ok {
			return nil, fmt.Errorf("cannot cast raw data to float")
		}
		return math.Round(castedRaw), nil
	case "DIV":
		castedRaw, ok := raw.(float64)
		if !ok {
			return nil, fmt.Errorf("cannot cast raw data to float")
		}
		return castedRaw / reducer.Args.(float64), nil
	case "DIVFROM":
		castedRaw, ok := raw.(float64)
		if !ok {
			return nil, fmt.Errorf("cannot cast raw data to float")
		}
		return reducer.Args.(float64) / castedRaw, nil
	default:
		return nil, fmt.Errorf("unknown reducer function: %s", reducer.Function)
	}

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
