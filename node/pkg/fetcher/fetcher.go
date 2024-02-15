package fetcher

import (
	"encoding/json"

	"bisonai.com/orakl/node/pkg/bus"
)

type Feed struct {
	FeedName   string          `json:"feedName" db:"name"`
	Definition json.RawMessage `json:"definition" db:"definition"`
}

type Adapter struct {
	AdapterName string `json:"adapterName" db:"name"`
	Feeds       []Feed `json:"feeds"`
}

type Fetcher struct {
	Bus      *bus.MessageBus
	Adapters []Adapter
}

func NewFetcher(bus *bus.MessageBus) *Fetcher {
	return &Fetcher{
		Adapters: make([]Adapter, 0),
		Bus:      bus,
	}
}

func (f *Fetcher) Run() error {
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
