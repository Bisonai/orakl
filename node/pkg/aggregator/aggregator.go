package aggregator

import (
	"bisonai.com/orakl/node/pkg/bus"
)

func New(bus *bus.MessageBus) *App {
	return &App{
		Aggregators: make(map[int64]*AggregatorNode, 0),
		Bus:         bus,
	}
}
