package reporter

import (
	"context"
	"errors"

	"bisonai.com/orakl/node/pkg/bus"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

func New(bus *bus.MessageBus, h host.Host, ps *pubsub.PubSub) *App {
	return &App{
		Reporter: &ReporterNode{},
		Bus:      bus,
		Host:     h,
		Pubsub:   ps,
	}
}

func (a *App) Run(ctx context.Context) error {
	err := a.setReporter(ctx, a.Host, a.Pubsub)
	if err != nil {
		return err
	}

	a.subscribe(ctx)

	return a.startReporter(ctx)
}

func (a *App) setReporter(ctx context.Context, h host.Host, ps *pubsub.PubSub) error {
	reporter, err := NewNode(ctx, h, ps)
	if err != nil {
		return err
	}
	a.Reporter = reporter
	return nil
}

func (a *App) startReporter(ctx context.Context) error {
	if a.Reporter.isRunning {
		log.Debug().Str("Player", "Reporter").Msg("reporter already running")
		return errors.New("reporter already running")
	}

	err := a.Reporter.SetKlaytnHelper(ctx)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to set klaytn helper")
		return err
	}

	nodeCtx, cancel := context.WithCancel(ctx)
	a.Reporter.nodeCtx = nodeCtx
	a.Reporter.nodeCancel = cancel
	a.Reporter.isRunning = true

	go a.Reporter.Run(nodeCtx)
	return nil
}

func (a *App) stopReporter() error {
	if !a.Reporter.isRunning {
		log.Debug().Str("Player", "Reporter").Msg("reporter not running")
		return errors.New("reporter not running")
	}

	a.Reporter.nodeCancel()
	a.Reporter.isRunning = false
	a.Reporter.KlaytnHelper.Close()
	return nil
}

func (a *App) subscribe(ctx context.Context) {
	log.Debug().Str("Player", "Reporter").Msg("subscribing to reporter topic")
	channel := a.Bus.Subscribe(bus.REPORTER)
	go func() {
		log.Debug().Str("Player", "Reporter").Msg("start reporter subscription goroutine")
		for {
			select {
			case msg := <-channel:
				log.Debug().Str("Player", "Reporter").Str("command", msg.Content.Command).Msg("received message from reporter topic")
				go a.handleMessage(ctx, msg)
			case <-ctx.Done():
				log.Debug().Str("Player", "Reporter").Msg("stopping reporter subscription goroutine")
				return
			}
		}
	}()
}

func (a *App) handleMessage(ctx context.Context, msg bus.Message) {
	switch msg.Content.Command {
	case bus.ACTIVATE_REPORTER:
		if msg.From != bus.ADMIN {
			bus.HandleMessageError(errors.New("non-admin"), msg, "reporter received message from non-admin")
			return
		}
		err := a.startReporter(ctx)
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to start reporter")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.DEACTIVATE_REPORTER:
		if msg.From != bus.ADMIN {
			bus.HandleMessageError(errors.New("non-admin"), msg, "reporter received message from non-admin")
			return
		}
		err := a.stopReporter()
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to stop reporter")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.REFRESH_REPORTER:
		if msg.From != bus.ADMIN {
			bus.HandleMessageError(errors.New("non-admin"), msg, "reporter received message from non-admin")
			return
		}
		err := a.stopReporter()
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to stop reporter")
			return
		}

		err = a.startReporter(ctx)
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to start reporter")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	}

}
