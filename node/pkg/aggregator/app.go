package aggregator

import (
	"context"
	"math/rand"
	"strconv"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"

	errorSentinel "bisonai.com/orakl/node/pkg/error"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

func New(bus *bus.MessageBus, h host.Host, ps *pubsub.PubSub) *App {
	return &App{
		Aggregators: make(map[int32]*Aggregator),
		Bus:         bus,
		Host:        h,
		Pubsub:      ps,
	}
}

func (a *App) Run(ctx context.Context) error {
	err := a.setAggregators(ctx, a.Host, a.Pubsub)
	if err != nil {
		return err
	}

	a.subscribe(ctx)

	return a.startAllAggregators(ctx)
}

func (a *App) setAggregators(ctx context.Context, h host.Host, ps *pubsub.PubSub) error {
	err := a.clearAggregators()
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to clear aggregators in setAggregators method")
		return err
	}

	loadedAggregators, err := a.getConfigs(ctx)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to get aggregator configs")
		return err
	}

	return a.initializeLoadedAggregators(loadedAggregators, h, ps)
}

func (a *App) clearAggregators() error {
	if a.Aggregators == nil {
		return nil
	}
	for _, aggregator := range a.Aggregators {
		if aggregator.isRunning {
			err := a.stopAggregator(aggregator)
			if err != nil {
				log.Error().Str("Player", "Aggregator").Err(err).Msgf("Failed to stop aggregator with ID: %d", aggregator.ID)
				return err
			}
		}
	}
	a.Aggregators = make(map[int32]*Aggregator)
	return nil
}

func (a *App) initializeLoadedAggregators(loadedAggregators []Config, h host.Host, ps *pubsub.PubSub) error {
	for _, aggregator := range loadedAggregators {
		if a.Aggregators[aggregator.ID] != nil {
			continue
		}

		topicString := aggregator.Name + "-global-aggregator-topic-" + strconv.Itoa(int(aggregator.AggregateInterval))
		tmpNode, err := NewAggregator(h, ps, topicString, aggregator)
		if err != nil {
			return err
		}
		a.Aggregators[aggregator.ID] = tmpNode

	}
	return nil
}

func (a *App) getConfigs(ctx context.Context) ([]Config, error) {
	return db.QueryRows[Config](ctx, SelectConfigQuery, nil)
}

func (a *App) startAggregator(ctx context.Context, aggregator *Aggregator) error {
	if aggregator == nil {
		return errorSentinel.ErrAggregatorNotFound
	}

	log.Debug().Str("Player", "Aggregator").Str("name", aggregator.Name).Msg("starting aggregator")
	if aggregator.isRunning {
		log.Debug().Str("Player", "Aggregator").Str("name", aggregator.Name).Msg("aggregator already running")
		return nil
	}

	nodeCtx, cancel := context.WithCancel(ctx)
	aggregator.nodeCtx = nodeCtx
	aggregator.nodeCancel = cancel
	aggregator.isRunning = true

	go aggregator.Run(nodeCtx)
	log.Info().Str("Player", "Aggregator").Str("name", aggregator.Name).Msg("Aggregator started successfully")
	return nil
}

func (a *App) startAggregatorById(ctx context.Context, id int32) error {
	aggregator, ok := a.Aggregators[id]
	if !ok {
		return errorSentinel.ErrAggregatorNotFound
	}
	return a.startAggregator(ctx, aggregator)
}

func (a *App) startAllAggregators(ctx context.Context) error {
	for _, aggregator := range a.Aggregators {
		err := a.startAggregator(ctx, aggregator)
		if err != nil {
			log.Error().Str("Player", "Aggregator").Err(err).Str("name", aggregator.Name).Msg("failed to start aggregator")
			return err
		}
		// starts with random sleep to avoid all aggregators starting at the same time
		jitter := time.Duration(rand.Intn(100)) * time.Millisecond
		time.Sleep(time.Millisecond*200 + jitter)
	}
	return nil
}

func (a *App) stopAggregator(aggregator *Aggregator) error {
	log.Debug().Str("Player", "Aggregator").Str("name", aggregator.Name).Msg("stopping aggregator")
	if !aggregator.isRunning {
		log.Debug().Str("Player", "Aggregator").Str("name", aggregator.Name).Msg("aggregator already stopped")
		return nil
	}
	if aggregator.nodeCancel == nil {
		return errorSentinel.ErrAggregatorCancelNotFound
	}
	aggregator.nodeCancel()
	aggregator.isRunning = false
	<-aggregator.nodeCtx.Done()
	return nil
}

func (a *App) stopAggregatorById(id int32) error {
	aggregator, ok := a.Aggregators[id]
	if !ok {
		return errorSentinel.ErrAggregatorNotFound
	}
	return a.stopAggregator(aggregator)
}

func (a *App) stopAllAggregators() error {
	for _, aggregator := range a.Aggregators {
		err := a.stopAggregator(aggregator)
		if err != nil {
			log.Error().Str("Player", "Aggregator").Err(err).Str("name", aggregator.Name).Msg("failed to stop aggregator")
			return err
		}
	}
	return nil
}

func (a *App) subscribe(ctx context.Context) {
	log.Debug().Str("Player", "Aggregator").Msg("subscribing to aggregator topics")
	channel := a.Bus.Subscribe(bus.AGGREGATOR)
	go func() {
		log.Debug().Str("Player", "Aggregator").Msg("starting aggregator subscription goroutine")
		for {
			select {
			case msg := <-channel:
				log.Debug().
					Str("Player", "Aggregator").
					Str("from", msg.From).
					Str("to", msg.To).
					Str("command", msg.Content.Command).
					Msg("fetcher received bus message")
				go a.handleMessage(ctx, msg)
			case <-ctx.Done():
				log.Debug().Str("Player", "Aggregator").Msg("stopping aggregator subscription goroutine")
				return
			}
		}
	}()
}

func (a *App) getAggregatorByName(name string) (*Aggregator, error) {
	for _, aggregator := range a.Aggregators {
		if aggregator.Name == name {
			return aggregator, nil
		}
	}
	return nil, errorSentinel.ErrAggregatorNotFound
}

func (a *App) handleMessage(ctx context.Context, msg bus.Message) {
	// TODO: Consider refactoring the handleMessage method to improve its structure and readability. Using a switch-case with many cases can be simplified by mapping commands to handler functions.

	// expected messages
	// from admin: control related messages, activate and deactivate aggregator

	if msg.To != bus.AGGREGATOR {
		log.Debug().Str("Player", "Aggregator").Msg("message not for aggregator")
		return
	}

	if msg.From != bus.ADMIN {
		bus.HandleMessageError(errorSentinel.ErrBusNonAdmin, msg, "aggregator received message from non-admin")
		return
	}

	switch msg.Content.Command {
	case bus.ACTIVATE_AGGREGATOR:
		log.Debug().Str("Player", "Aggregator").Msg("activate aggregator msg received")
		aggregatorId, err := bus.ParseInt32MsgParam(msg, "id")
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to parse aggregatorId")
			return
		}

		log.Debug().Str("Player", "Aggregator").Int32("aggregatorId", aggregatorId).Msg("activating aggregator")
		err = a.startAggregatorById(ctx, aggregatorId)
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to start aggregator")
			return
		}
		log.Debug().Str("Player", "Aggregator").Msg("sending success response for activate aggregator")
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.DEACTIVATE_AGGREGATOR:
		log.Debug().Str("Player", "Aggregator").Msg("deactivate aggregator msg received")
		aggregatorId, err := bus.ParseInt32MsgParam(msg, "id")
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to parse aggregatorId")
			return
		}

		log.Debug().Str("Player", "Aggregator").Int32("aggregatorId", aggregatorId).Msg("deactivating aggregator")
		err = a.stopAggregatorById(aggregatorId)
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to stop aggregator")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.REFRESH_AGGREGATOR_APP:
		log.Debug().Str("Player", "Aggregator").Msg("refresh aggregator msg received")
		err := a.stopAllAggregators()
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to stop all aggregators")
			return
		}
		err = a.setAggregators(ctx, a.Host, a.Pubsub)
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to set aggregators")
			return
		}
		err = a.startAllAggregators(ctx)
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to start all aggregators")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.STOP_AGGREGATOR_APP:
		log.Debug().Str("Player", "Aggregator").Msg("stop aggregator msg received")
		err := a.stopAllAggregators()
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to stop all aggregators")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.START_AGGREGATOR_APP:
		log.Debug().Str("Player", "Aggregator").Msg("start aggregator msg received")
		err := a.startAllAggregators(ctx)
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to start all aggregators")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	default:
		bus.HandleMessageError(errorSentinel.ErrBusUnknownCommand, msg, "aggregator received unknown command")
		return
	}
}
