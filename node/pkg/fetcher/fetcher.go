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

const FETCHER_FREQUENCY = 2 * time.Second

func NewFetcher(bus *bus.MessageBus) *Fetcher {
	return &Fetcher{
		Adapters: make([]AdapterDetail, 0),
		Bus:      bus,
	}
}

func (f *Fetcher) SubscribeMsg(ctx context.Context) {
	channel := f.Bus.Subscribe("fetcher", 10)
	go func() {
		msg := <-channel
		err := f.MessageHandler(ctx, msg)
		if err != nil {
			fmt.Println(err)
		}
	}()
}

func (f *Fetcher) MessageHandler(ctx context.Context, msg bus.Message) error {
	switch msg.Content.Command {
	case "start":
		if msg.From != "admin" {
			return fmt.Errorf("only admin can start")
		}
		return f.Run(ctx)
	case "stop":
		if msg.From != "admin" {
			return fmt.Errorf("only admin can stop")
		}
		return f.Stop()
	case "refresh":
		if msg.From != "admin" {
			return fmt.Errorf("only admin can refresh")
		}
		return f.initialize(ctx)
	}
	return nil
}

func (f *Fetcher) Run(ctx context.Context) error {
	if f.running {
		return fmt.Errorf("fetcher already running")
	}

	f.fetcherCtx, f.cancel = context.WithCancel(ctx)
	f.running = true

	err := f.initialize(ctx)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(FETCHER_FREQUENCY)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				err := f.runAdapter(f.fetcherCtx)
				if err != nil {
					fmt.Println(err)
				}
			case <-f.fetcherCtx.Done():
				return
			}
		}
	}()

	return nil
}

func (f *Fetcher) Stop() error {
	if !f.running {
		return fmt.Errorf("fetcher not running")
	}

	if f.cancel != nil {
		f.cancel()
	}
	f.running = false
	return nil

}

func (f *Fetcher) runAdapter(ctx context.Context) error {
	for _, adapter := range f.Adapters {
		result, err := f.fetch(adapter)
		if err != nil {
			return err
		}
		aggregated := utils.GetFloatAvg(result)
		err = f.insertPgsql(ctx, adapter.Name, aggregated)
		if err != nil {
			return err
		}
		err = f.insertRdb(ctx, adapter.Name, aggregated)
		if err != nil {
			return err
		}
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
			fmt.Println(err)
			continue
		}
		res, err := utils.GetRequest[interface{}](definition.Url, nil, definition.Headers)
		if err != nil {
			fmt.Println(err)
			continue
		}

		result, err := utils.Reduce(res, definition.Reducers)
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
