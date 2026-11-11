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
// the last value repeats until the peer is back or the boot API drops it.
var defaultReconnectBackoff = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	5 * time.Second,
	10 * time.Second,
	30 * time.Second,
	60 * time.Second,
}

// floor between boot API syncs, also across separate disconnect events
const defaultMinSyncInterval = 1 * time.Second

type App struct {
	Host host.Host
	Bus  *bus.MessageBus

	// reconnect state; timing is injectable for tests
	reconnect       func(context.Context, host.Host) ([]peer.ID, error)
	backoff         []time.Duration
	minSyncInterval time.Duration

	mu      sync.Mutex
	pending map[peer.ID]struct{}
	newPeer bool // a not-yet-pending peer was queued since the worker last looked
	wake    chan struct{}
}

func New(bus *bus.MessageBus, h host.Host) *App {
	return &App{
		Bus:             bus,
		Host:            h,
		reconnect:       setup.SyncThroughBootApi,
		backoff:         defaultReconnectBackoff,
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
		defer sub.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case e, ok := <-sub.Out():
				if !ok {
					return
				}
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
	if _, ok := a.pending[evt.Peer]; !ok {
		a.pending[evt.Peer] = struct{}{}
		a.newPeer = true
	}
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
// backoff, until every pending peer is connected again or no longer listed by
// the boot API, or ctx is done. A newly disconnected peer restarts the backoff.
// It returns the time of the last sync.
func (a *App) reconnectUntilConnected(ctx context.Context, lastSync time.Time) time.Time {
	attempt := 0
	a.takeNewPeer()
	for {
		if a.prunePending(nil, false) == 0 {
			return lastSync
		}
		if wait := a.minSyncInterval - time.Since(lastSync); wait > 0 {
			if !sleepCtx(ctx, wait) {
				return lastSync
			}
		}

		known, err := a.reconnect(ctx, a.Host)
		lastSync = time.Now()
		attempt++
		if err != nil {
			log.Error().Err(err).Str("Player", "Libp2pHelper").Int("attempt", attempt).Msg("Error occurred on boot API sync")
		}

		if a.prunePending(known, err == nil) == 0 {
			log.Info().Str("Player", "Libp2pHelper").Int("attempt", attempt).Msg("peers reconnected")
			return lastSync
		}

		reset, ok := a.waitBackoff(ctx, a.backoffAfter(attempt))
		if !ok {
			return lastSync
		}
		if reset {
			attempt = 0 // newly lost peer: retry now, not after backoff
		}
	}
}

// waitBackoff waits d, cut short only when a new peer is queued (reset).
// Repeat events for already pending peers keep waiting. ok is false on ctx done.
func (a *App) waitBackoff(ctx context.Context, d time.Duration) (reset, ok bool) {
	t := time.NewTimer(d)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return false, false
		case <-t.C:
			return false, true
		case <-a.wake:
			if a.takeNewPeer() {
				return true, true
			}
		}
	}
}

// prunePending drops pending peers that are connected again and, after a
// successful sync, peers the boot API does not list (e.g. the old ID of a
// restarted node, or a non-oracle peer). It returns how many remain.
func (a *App) prunePending(known []peer.ID, syncOK bool) int {
	listed := make(map[peer.ID]struct{}, len(known))
	for _, p := range known {
		listed[p] = struct{}{}
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	for p := range a.pending {
		_, ok := listed[p]
		if a.Host.Network().Connectedness(p) == network.Connected || (syncOK && !ok) {
			delete(a.pending, p)
		}
	}
	return len(a.pending)
}

// backoffAfter returns the wait after the given (1-based) failed attempt.
func (a *App) backoffAfter(attempt int) time.Duration {
	b := a.backoff
	if len(b) == 0 {
		b = defaultReconnectBackoff
	}
	return b[min(max(attempt-1, 0), len(b)-1)]
}

// takeNewPeer reports and clears whether a new peer was queued.
func (a *App) takeNewPeer() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	n := a.newPeer
	a.newPeer = false
	return n
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
