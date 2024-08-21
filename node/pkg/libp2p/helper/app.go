package helper

import (
	"context"
	"fmt"
	"time"

	"bisonai.com/miko/node/pkg/bus"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/libp2p/setup"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/rs/zerolog/log"
)

type App struct {
	Host host.Host
	Bus  *bus.MessageBus
}

func New(bus *bus.MessageBus, h host.Host) *App {
	return &App{
		Bus:  bus,
		Host: h,
	}
}

func (a *App) Run(ctx context.Context) error {
	defer a.subscribe(ctx)

	sub, err := a.Host.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
	if err != nil {
		return fmt.Errorf("event subscription failed: %w", err)
	}

	a.subscribeLibp2pEvent(ctx, sub)
	return nil
}

func (a *App) subscribe(ctx context.Context) {
	log.Debug().Str("Player", "Libp2pHelper").Msg("subscribing to libp2pHelper topics")
	channel := a.Bus.Subscribe(bus.LIBP2P)
	go func() {
		log.Debug().Str("Player", "Libp2pHelper").Msg("starting libp2p subscription goroutine")
		for {
			select {
			case msg := <-channel:
				log.Debug().
					Str("Player", "Libp2pHelper").
					Str("from", msg.From).
					Str("to", msg.To).
					Str("command", msg.Content.Command).
					Msg("libp2p received bus message")
				go a.handleMessage(ctx, msg)
			case <-ctx.Done():
				log.Debug().Str("Player", "Libp2pHelper").Msg("stopping libp2pHelper subscription goroutine")
				return
			}
		}
	}()
}

func (a *App) handleMessage(ctx context.Context, msg bus.Message) {
	if msg.To != bus.LIBP2P {
		log.Debug().Str("Player", "Libp2pHelper").Msg("message not for libp2pHelper")
		return
	}
	if msg.From != bus.ADMIN {
		bus.HandleMessageError(errorSentinel.ErrBusNonAdmin, msg, "libp2pHelper received message from non-admin")
	}

	switch msg.Content.Command {
	case bus.GET_PEER_COUNT:
		log.Debug().Str("Player", "Libp2pHelper").Msg("get peer count msg received")
		peerCount := len(a.Host.Network().Peers())
		msg.Response <- bus.MessageResponse{Success: true, Args: map[string]any{"Count": peerCount}}
	case bus.LIBP2P_SYNC:
		log.Debug().Str("Player", "Libp2pHelper").Msg("libp2p sync msg received")
		err := setup.ConnectThroughBootApi(ctx, a.Host)
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to sync through boot api")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	default:
		bus.HandleMessageError(errorSentinel.ErrBusUnknownCommand, msg, "libp2p helper received unknown command")
		return
	}
}

func (a *App) subscribeLibp2pEvent(ctx context.Context, sub event.Subscription) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				sub.Close()
				return
			case e := <-sub.Out():
				a.handleDisconnectEvent(ctx, e)
			}
		}
	}()
}

func (a *App) handleDisconnectEvent(ctx context.Context, e interface{}) {
	log.Info().Str("Player", "Libp2pHelper").Msg("Disconnect event caught, triggering resync")
	evt := e.(event.EvtPeerConnectednessChanged)
	if evt.Connectedness == network.NotConnected {
		for i := 1; i < 4; i++ {
			// do not attempt immediate resync, but give some time
			time.Sleep(time.Duration(i) * time.Minute)
			err := setup.ConnectThroughBootApi(ctx, a.Host)
			if err != nil {
				log.Error().Err(err).Str("Player", "Libp2pHelper").Msg("Error occurred on boot API sync")
			}
		}
	}
}
