package aggregator

import (
	"context"
	"errors"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"

	"bisonai.com/orakl/node/pkg/utils"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

func New(bus *bus.MessageBus) *App {
	return &App{
		Aggregators: make(map[int64]*AggregatorNode, 0),
		Bus:         bus,
	}
}

func (a *App) Run(ctx context.Context, h host.Host, ps *pubsub.PubSub) error {
	err := a.initialize(ctx, h, ps)
	if err != nil {
		return err
	}

	a.subscribe(ctx)

	for _, aggregator := range a.Aggregators {
		err = a.startAggregator(ctx, aggregator)
		if err != nil {
			log.Error().Err(err).Str("name", aggregator.Name).Msg("failed to start aggregator")
		}
	}

	return nil
}

func (a *App) initialize(ctx context.Context, h host.Host, ps *pubsub.PubSub) error {
	aggregators, err := a.getAggregators(ctx)
	if err != nil {
		return err
	}
	a.Aggregators = make(map[int64]*AggregatorNode, len(aggregators))

	for _, aggregator := range aggregators {
		topicString, err := utils.EncryptText(aggregator.Name)
		if err != nil {
			return err
		}

		tmpNode, err := NewNode(h, ps, topicString)
		if err != nil {
			return err
		}
		tmpNode.Aggregator = aggregator
		a.Aggregators[aggregator.ID] = tmpNode
	}
	return nil
}

func (a *App) getAggregators(ctx context.Context) ([]Aggregator, error) {
	return db.QueryRows[Aggregator](ctx, SelectActiveAggregatorsQuery, nil)
}

func (a *App) startAggregator(ctx context.Context, aggregator *AggregatorNode) error {
	log.Debug().Str("name", aggregator.Name).Msg("starting aggregator")
	if aggregator.isRunning {
		log.Debug().Str("name", aggregator.Name).Msg("aggregator already running")
		return errors.New("aggregator already running")
	}

	nodeCtx, cancel := context.WithCancel(ctx)
	aggregator.nodeCtx = nodeCtx
	aggregator.nodeCancel = cancel
	aggregator.isRunning = true

	return aggregator.Run(ctx)
}

func (a *App) startAggregatorById(ctx context.Context, id int64) error {
	aggregator, ok := a.Aggregators[id]
	if !ok {
		return errors.New("aggregator not found")
	}
	return a.startAggregator(ctx, aggregator)
}

func (a *App) stopAggregator(ctx context.Context, aggregator *AggregatorNode) error {
	log.Debug().Str("name", aggregator.Name).Msg("stopping aggregator")
	if !aggregator.isRunning {
		log.Debug().Str("name", aggregator.Name).Msg("aggregator already stopped")
		return errors.New("aggregator already stopped")
	}
	if aggregator.nodeCancel == nil {
		return errors.New("aggregator cancel function not found")
	}
	aggregator.nodeCancel()
	aggregator.isRunning = false
	return nil
}

func (a *App) stopAggregatorById(ctx context.Context, id int64) error {
	aggregator, ok := a.Aggregators[id]
	if !ok {
		return errors.New("aggregator not found")
	}
	return a.stopAggregator(ctx, aggregator)
}

func (a *App) subscribe(ctx context.Context) {
	log.Debug().Msg("subscribing to aggregator topics")
	channel := a.Bus.Subscribe(bus.AGGREGATOR)
	go func() {
		log.Debug().Msg("starting aggregator subscription goroutine")
		for {
			select {
			case msg := <-channel:
				log.Debug().
					Str("from", msg.From).
					Str("to", msg.To).
					Str("command", msg.Content.Command).
					Msg("fetcher received message")
				a.handleMessage(ctx, msg)
			case <-ctx.Done():
				log.Debug().Msg("stopping aggregator subscription goroutine")
				return
			}
		}
	}()
}

func (a *App) getAggregatorByName(name string) (*AggregatorNode, error) {
	for _, aggregator := range a.Aggregators {
		if aggregator.Name == name {
			return aggregator, nil
		}
	}
	return nil, errors.New("aggregator not found")
}

func (a *App) handleMessage(ctx context.Context, msg bus.Message) {
	// expected messages
	// from admin: control related messages, activate and deactivate aggregator
	// from fetcher: deviation related messages

	if msg.To != bus.AGGREGATOR {
		log.Debug().Msg("message not for aggregator")
		return
	}

	switch msg.Content.Command {
	case bus.ACTIVATE_AGGREGATOR:
		if msg.From != bus.ADMIN {
			log.Debug().Msg("aggregator received message from non-admin")
			return
		}
		log.Debug().Msg("activate aggregator msg received")
		aggregatorId, err := bus.ParseInt64MsgParam(msg, "id")
		if err != nil {
			log.Error().Err(err).Msg("failed to parse aggregatorId")
			return
		}

		log.Debug().Int64("aggregatorId", aggregatorId).Msg("activating aggregator")
		err = a.startAggregatorById(ctx, aggregatorId)
		if err != nil {
			log.Error().Err(err).Msg("failed to start aggregator")
		}
	case bus.DEACTIVATE_AGGREGATOR:
		if msg.From != bus.ADMIN {
			log.Debug().Msg("aggregator received message from non-admin")
			return
		}
		log.Debug().Msg("deactivate aggregator msg received")
		aggregatorId, err := bus.ParseInt64MsgParam(msg, "id")
		if err != nil {
			log.Error().Err(err).Msg("failed to parse aggregatorId")
			return
		}

		log.Debug().Int64("aggregatorId", aggregatorId).Msg("deactivating aggregator")
		err = a.stopAggregatorById(ctx, aggregatorId)
		if err != nil {
			log.Error().Err(err).Msg("failed to stop aggregator")
		}
	case bus.REFRESH_AGGREGATOR_APP:
		// TODO: refresh aggregator
		log.Debug().Msg("refresh aggregator msg received")
	case bus.STOP_AGGREGATOR_APP:
		// TODO: stop aggregator
		log.Debug().Msg("stop aggregator msg received")
	case bus.START_AGGREGATOR_APP:
		// TODO: start aggregator
		log.Debug().Msg("start aggregator msg received")

	case bus.DEVIATION:
		if msg.From != bus.FETCHER {
			log.Debug().Msg("aggregator received deviation message from non-fetcher")
			return
		}
		log.Debug().Msg("deviation message received")
		aggregatorName, err := bus.ParseStringMsgParam(msg, "name")
		if err != nil {
			log.Error().Err(err).Msg("failed to parse aggregator name")
			return
		}
		aggregator, err := a.getAggregatorByName(aggregatorName)
		if err != nil {
			log.Error().Err(err).Msg("aggregator not found")
			return
		}

		err = aggregator.executeDeviation()
		if err != nil {
			log.Error().Err(err).Msg("failed to execute deviation")
		}
	}
}
