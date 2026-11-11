package helper

import (
	"context"
	"fmt"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/bus"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/libp2p/setup"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
)

// defaultReconnectBackoff is the wait after each failed reconnect attempt;
// the last value repeats until defaultMaxReconnectAttempts.
var defaultReconnectBackoff = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	5 * time.Second,
	10 * time.Second,
	30 * time.Second,
	60 * time.Second,
}

const (
	// ~10 min of retrying before waiting for the next disconnect event
	defaultMaxReconnectAttempts = 15
	// floor between boot API syncs, also across separate disconnect events
	defaultMinSyncInterval = 1 * time.Second
)

type App struct {
	Host host.Host
	Bus  *bus.MessageBus

	// reconnect state; timing is injectable for tests
	reconnect       func(context.Context, host.Host) error
	backoff         []time.Duration
	maxAttempts     int
	minSyncInterval time.Duration

	mu      sync.Mutex
	pending map[peer.ID]struct{}
	wake    chan struct{}
}

func New(bus *bus.MessageBus, h host.Host) *App {
	return &App{
		Bus:             bus,
		Host:            h,
		reconnect:       setup.ConnectThroughBootApi,
		backoff:         defaultReconnectBackoff,
		maxAttempts:     defaultMaxReconnectAttempts,
		minSyncInterval: defaultMinSyncInterval,
		pending:         make(map[peer.ID]struct{}),
		wake:            make(chan struct{}, 1),
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
	go a.reconnectWorker(ctx)
	go func() {
		for {
			select {
			case <-ctx.Done():
				sub.Close()
				return
			case e := <-sub.Out():
				a.handleDisconnectEvent(e)
			}
		}
	}()
}

// handleDisconnectEvent only queues the peer and wakes the reconnect worker;
// it never blocks the event loop.
func (a *App) handleDisconnectEvent(e interface{}) {
	evt, ok := e.(event.EvtPeerConnectednessChanged)
	if !ok || evt.Connectedness != network.NotConnected {
		return
	}
	log.Info().Str("Player", "Libp2pHelper").Str("peer", evt.Peer.String()).Msg("Disconnect event caught, triggering resync")

	a.mu.Lock()
	a.pending[evt.Peer] = struct{}{}
	a.mu.Unlock()

	select {
	case a.wake <- struct{}{}:
	default: // worker already signalled; event coalesced
	}
}

// reconnectWorker is the single per-process reconnect loop.
func (a *App) reconnectWorker(ctx context.Context) {
	var lastSync time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-a.wake:
		}
		lastSync = a.reconnectUntilConnected(ctx, lastSync)
	}
}

// reconnectUntilConnected syncs through the boot API right away, then with
// backoff, until every pending peer is connected again, maxAttempts is hit or
// ctx is done. It returns the time of the last sync.
func (a *App) reconnectUntilConnected(ctx context.Context, lastSync time.Time) time.Time {
	for attempt := 0; ; attempt++ {
		if a.prunePending() == 0 {
			return lastSync
		}
		if wait := a.minSyncInterval - time.Since(lastSync); wait > 0 {
			if !sleepCtx(ctx, wait) {
				return lastSync
			}
		}

		err := a.reconnect(ctx, a.Host)
		lastSync = time.Now()
		if err != nil {
			log.Error().Err(err).Str("Player", "Libp2pHelper").Int("attempt", attempt+1).Msg("Error occurred on boot API sync")
		}

		if a.prunePending() == 0 {
			log.Info().Str("Player", "Libp2pHelper").Int("attempt", attempt+1).Msg("peers reconnected")
			return lastSync
		}
		if attempt+1 >= a.maxAttempts {
			log.Warn().Str("Player", "Libp2pHelper").Int("attempts", attempt+1).Msg("giving up reconnect until next disconnect event")
			a.clearPending()
			return lastSync
		}

		if !sleepCtx(ctx, a.backoff[min(attempt, len(a.backoff)-1)]) {
			return lastSync
		}
	}
}

// prunePending drops pending peers that are connected again and returns how
// many are still disconnected.
func (a *App) prunePending() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	for p := range a.pending {
		if a.Host.Network().Connectedness(p) == network.Connected {
			delete(a.pending, p)
		}
	}
	return len(a.pending)
}

func (a *App) clearPending() {
	a.mu.Lock()
	defer a.mu.Unlock()
	clear(a.pending)
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
