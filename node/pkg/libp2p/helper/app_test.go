package helper

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/bus"
	"bisonai.com/miko/node/pkg/libp2p/setup"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

type callRecorder struct {
	mu    sync.Mutex
	times []time.Time
}

func (r *callRecorder) add() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.times = append(r.times, time.Now())
	return len(r.times)
}

func (r *callRecorder) snapshot() []time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]time.Time(nil), r.times...)
}

type reconnectFunc = func(context.Context, host.Host) ([]peer.ID, error)

// knows returns a reconnect fake that records the call and reports ids as
// known to the boot API without dialing.
func (r *callRecorder) knows(ids ...peer.ID) reconnectFunc {
	return func(context.Context, host.Host) ([]peer.ID, error) {
		r.add()
		return ids, nil
	}
}

func newHost(t *testing.T) host.Host {
	t.Helper()
	h, err := setup.NewHost(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = h.Close() })
	return h
}

// connectedPair returns a running App on h1, connected to h2; mk builds the
// reconnect fake for h2.
func connectedPair(t *testing.T, ctx context.Context, mk func(h2 host.Host) reconnectFunc, backoff []time.Duration) (*App, host.Host, host.Host) {
	t.Helper()
	h1, h2 := newHost(t), newHost(t)

	app := New(bus.New(10), h1)
	app.reconnect = mk(h2)
	app.backoff = backoff
	app.minSyncInterval = 0
	require.NoError(t, app.Run(ctx))

	require.NoError(t, h1.Connect(ctx, peer.AddrInfo{ID: h2.ID(), Addrs: h2.Addrs()}))
	return app, h1, h2
}

func dial(ctx context.Context, h, target host.Host) error {
	return h.Connect(ctx, peer.AddrInfo{ID: target.ID(), Addrs: target.Addrs()})
}

func TestReconnectImmediateAndStopsOnSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rec := &callRecorder{}
	_, h1, h2 := connectedPair(t, ctx, func(h2 host.Host) reconnectFunc {
		return func(ctx context.Context, h host.Host) ([]peer.ID, error) {
			rec.add()
			return []peer.ID{h2.ID()}, dial(ctx, h, h2)
		}
	}, []time.Duration{10 * time.Millisecond})

	start := time.Now()
	require.NoError(t, h1.Network().ClosePeer(h2.ID()))

	require.Eventually(t, func() bool { return len(rec.snapshot()) >= 1 }, 2*time.Second, 5*time.Millisecond)
	require.Less(t, rec.snapshot()[0].Sub(start), time.Second, "first attempt must be immediate")

	// connected after first attempt: no further syncs
	time.Sleep(100 * time.Millisecond)
	require.Len(t, rec.snapshot(), 1)
	require.Len(t, h1.Network().Peers(), 1)
}

func TestReconnectBacksOffUntilConnected(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	backoff := []time.Duration{20 * time.Millisecond, 40 * time.Millisecond, 80 * time.Millisecond}
	rec := &callRecorder{}
	_, h1, h2 := connectedPair(t, ctx, func(h2 host.Host) reconnectFunc {
		return func(ctx context.Context, h host.Host) ([]peer.ID, error) {
			if rec.add() < 4 {
				return []peer.ID{h2.ID()}, nil // sync "succeeds" but peer still down
			}
			return []peer.ID{h2.ID()}, dial(ctx, h, h2)
		}
	}, backoff)

	require.NoError(t, h1.Network().ClosePeer(h2.ID()))

	require.Eventually(t, func() bool { return len(rec.snapshot()) >= 4 }, 3*time.Second, 5*time.Millisecond)
	time.Sleep(300 * time.Millisecond)

	times := rec.snapshot()
	require.Len(t, times, 4, "worker must stop once reconnected")
	for i := 1; i < len(times); i++ {
		require.GreaterOrEqual(t, times[i].Sub(times[i-1]), backoff[i-1], "gap %d", i)
	}
	require.Len(t, h1.Network().Peers(), 1)
}

func TestDisconnectEventDoesNotBlockEventLoop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	release := make(chan struct{})
	rec := &callRecorder{}
	app := New(bus.New(10), newHost(t))
	app.reconnect = func(ctx context.Context, _ host.Host) ([]peer.ID, error) {
		rec.add()
		select {
		case <-release:
		case <-ctx.Done():
		}
		return nil, nil
	}
	app.minSyncInterval = 0
	go app.reconnectWorker(ctx)

	// a peer that is never connected keeps the worker busy in reconnect
	other := newHost(t)
	app.handleDisconnectEvent(evtNotConnected(other.ID()))
	require.Eventually(t, func() bool { return len(rec.snapshot()) == 1 }, time.Second, time.Millisecond)

	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			app.handleDisconnectEvent(evtNotConnected(other.ID()))
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handleDisconnectEvent blocked while reconnect in flight")
	}
	require.Len(t, rec.snapshot(), 1, "events must coalesce into the single worker")
	close(release)
}

func TestReconnectIgnoresNonDisconnectAndReturnedPeer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rec := &callRecorder{}
	h1, h2 := newHost(t), newHost(t)
	app := New(bus.New(10), h1)
	app.reconnect = rec.knows(h2.ID())
	app.minSyncInterval = 0
	go app.reconnectWorker(ctx)
	require.NoError(t, dial(ctx, h1, h2))

	app.handleDisconnectEvent(evtConnected(h2.ID()))
	// NotConnected for a peer that is connected again by the time the worker runs
	app.handleDisconnectEvent(evtNotConnected(h2.ID()))

	time.Sleep(100 * time.Millisecond)
	require.Empty(t, rec.snapshot())
}

func TestReconnectStopsOnCtxCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	rec := &callRecorder{}
	app := New(bus.New(10), newHost(t))
	other := newHost(t).ID()
	app.reconnect = rec.knows(other)
	app.backoff = []time.Duration{time.Hour}
	app.minSyncInterval = 0
	stopped := make(chan struct{})
	go func() { app.reconnectWorker(ctx); close(stopped) }()

	app.handleDisconnectEvent(evtNotConnected(other))
	require.Eventually(t, func() bool { return len(rec.snapshot()) == 1 }, time.Second, time.Millisecond)

	cancel()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop on ctx cancel")
	}
}

func evtNotConnected(p peer.ID) event.EvtPeerConnectednessChanged {
	return event.EvtPeerConnectednessChanged{Peer: p, Connectedness: network.NotConnected}
}

func evtConnected(p peer.ID) event.EvtPeerConnectednessChanged {
	return event.EvtPeerConnectednessChanged{Peer: p, Connectedness: network.Connected}
}

func TestReconnectDropsPeerUnknownToBootApi(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rec := &callRecorder{}
	app := New(bus.New(10), newHost(t))
	app.reconnect = rec.knows() // e.g. old ID of a restarted node, boot host
	app.backoff = []time.Duration{time.Millisecond}
	app.minSyncInterval = 0
	go app.reconnectWorker(ctx)

	app.handleDisconnectEvent(evtNotConnected(newHost(t).ID()))
	require.Eventually(t, func() bool { return len(rec.snapshot()) == unlistedSyncsToDrop }, time.Second, time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	require.Len(t, rec.snapshot(), unlistedSyncsToDrop, "peer unlisted by boot API must be dropped")
}

func TestReconnectToleratesStaleBootList(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := newHost(t).ID()
	rec := &callRecorder{}
	app := New(bus.New(10), newHost(t))
	app.reconnect = func(context.Context, host.Host) ([]peer.ID, error) {
		// listed only on every 3rd sync: misses never reach the drop threshold
		if rec.add()%unlistedSyncsToDrop == 0 {
			return []peer.ID{a}, nil
		}
		return nil, nil
	}
	app.backoff = []time.Duration{time.Millisecond}
	app.minSyncInterval = 0
	go app.reconnectWorker(ctx)

	app.handleDisconnectEvent(evtNotConnected(a))
	require.Eventually(t, func() bool { return len(rec.snapshot()) >= 4*unlistedSyncsToDrop }, 2*time.Second, time.Millisecond)
}

func TestReconnectSyncErrorKeepsPeer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rec := &callRecorder{}
	app := New(bus.New(10), newHost(t))
	app.reconnect = func(context.Context, host.Host) ([]peer.ID, error) {
		rec.add()
		return nil, errors.New("boot api down")
	}
	app.backoff = []time.Duration{time.Millisecond}
	app.minSyncInterval = 0
	go app.reconnectWorker(ctx)

	app.handleDisconnectEvent(evtNotConnected(newHost(t).ID()))
	require.Eventually(t, func() bool { return len(rec.snapshot()) >= 3 }, time.Second, time.Millisecond)
}

func TestNewDisconnectSkipsBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rec := &callRecorder{}
	a, b := newHost(t).ID(), newHost(t).ID()
	app := New(bus.New(10), newHost(t))
	app.reconnect = rec.knows(a, b)
	app.backoff = []time.Duration{time.Hour}
	app.minSyncInterval = 0
	go app.reconnectWorker(ctx)

	app.handleDisconnectEvent(evtNotConnected(a))
	require.Eventually(t, func() bool { return len(rec.snapshot()) == 1 }, time.Second, time.Millisecond)

	// worker now in 1h backoff for a; b must still be tried right away
	app.handleDisconnectEvent(evtNotConnected(b))
	require.Eventually(t, func() bool { return len(rec.snapshot()) == 2 }, time.Second, time.Millisecond)
}

func TestEmptyBackoffFallsBackToDefault(t *testing.T) {
	app := &App{}
	require.Equal(t, defaultReconnectBackoff[0], app.backoffAfter(1))
	app.backoff = []time.Duration{time.Millisecond, time.Second}
	require.Equal(t, time.Second, app.backoffAfter(10))
}

func TestReconnectKeepsRetryingAtCapUntilConnected(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rec := &callRecorder{}
	h1, h2 := newHost(t), newHost(t)
	app := New(bus.New(10), h1)
	app.reconnect = rec.knows(h2.ID())
	app.backoff = []time.Duration{time.Millisecond}
	app.minSyncInterval = 0
	require.NoError(t, app.Run(ctx))
	require.NoError(t, dial(ctx, h1, h2))

	require.NoError(t, h1.Network().ClosePeer(h2.ID()))
	require.Eventually(t, func() bool { return len(rec.snapshot()) >= 20 }, 2*time.Second, time.Millisecond, "no give-up while peer listed")

	require.NoError(t, dial(ctx, h1, h2))
	time.Sleep(50 * time.Millisecond)
	n := len(rec.snapshot())
	time.Sleep(50 * time.Millisecond)
	require.Equal(t, n, len(rec.snapshot()), "worker must stop once reconnected")
}

func TestNewPeerDuringSyncRetriedNow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a, b := newHost(t).ID(), newHost(t).ID()
	app := New(bus.New(10), newHost(t))
	rec := &callRecorder{}
	app.reconnect = func(context.Context, host.Host) ([]peer.ID, error) {
		if rec.add() == 1 {
			app.handleDisconnectEvent(evtNotConnected(b)) // b drops while syncing for a
		}
		return []peer.ID{a, b}, nil
	}
	app.backoff = []time.Duration{time.Hour}
	app.minSyncInterval = 0
	go app.reconnectWorker(ctx)

	app.handleDisconnectEvent(evtNotConnected(a))
	require.Eventually(t, func() bool { return len(rec.snapshot()) == 2 }, time.Second, time.Millisecond)
}

func TestRepeatEventForPendingPeerKeepsBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := newHost(t).ID()
	rec := &callRecorder{}
	app := New(bus.New(10), newHost(t))
	app.reconnect = rec.knows(a)
	app.backoff = []time.Duration{time.Hour}
	app.minSyncInterval = 0
	go app.reconnectWorker(ctx)

	app.handleDisconnectEvent(evtNotConnected(a))
	require.Eventually(t, func() bool { return len(rec.snapshot()) == 1 }, time.Second, time.Millisecond)
	for i := 0; i < 10; i++ {
		app.handleDisconnectEvent(evtNotConnected(a))
	}
	time.Sleep(50 * time.Millisecond)
	require.Len(t, rec.snapshot(), 1, "repeat event for same pending peer must not skip backoff")
}

func TestMinSyncIntervalAcrossEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rec := &callRecorder{}
	app := New(bus.New(10), newHost(t))
	app.reconnect = rec.knows() // peer unlisted: loop ends after unlistedSyncsToDrop syncs
	app.backoff = []time.Duration{time.Millisecond}
	app.minSyncInterval = 100 * time.Millisecond
	go app.reconnectWorker(ctx)

	other := newHost(t).ID()
	app.handleDisconnectEvent(evtNotConnected(other))
	require.Eventually(t, func() bool { return len(rec.snapshot()) == unlistedSyncsToDrop }, 2*time.Second, time.Millisecond)
	time.Sleep(20 * time.Millisecond) // let worker go idle
	app.handleDisconnectEvent(evtNotConnected(other))
	require.Eventually(t, func() bool { return len(rec.snapshot()) == unlistedSyncsToDrop+1 }, 2*time.Second, time.Millisecond)

	times := rec.snapshot()
	for i := 1; i < len(times); i++ {
		require.GreaterOrEqual(t, times[i].Sub(times[i-1]), app.minSyncInterval, "gap %d", i)
	}
}

// Closing the host through CloseHost drops its connections, but the worker
// doesn't try to reconnect: a closed host can't (#2817).
func TestCloseHostStopsReconnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rec := &callRecorder{}
	app, h1, h2 := connectedPair(t, ctx, func(h2 host.Host) reconnectFunc { return rec.knows(h2.ID()) },
		[]time.Duration{10 * time.Millisecond})
	require.Eventually(t, func() bool { return h2.Network().Connectedness(h1.ID()) == network.Connected },
		time.Second, 5*time.Millisecond)

	require.NoError(t, app.CloseHost())
	require.Eventually(t, func() bool { return h2.Network().Connectedness(h1.ID()) != network.Connected },
		time.Second, 5*time.Millisecond, "close did not drop the connection")
	time.Sleep(200 * time.Millisecond) // many backoffs
	require.Empty(t, rec.snapshot(), "reconnect tried after the host closed")
	require.NoError(t, app.CloseHost(), "second close")
}

// CloseHost before Run (a shutdown during startup) leaves Run with no worker.
func TestCloseHostBeforeRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rec := &callRecorder{}
	app := New(bus.New(10), newHost(t))
	app.reconnect = rec.knows()
	app.minSyncInterval = 0
	require.NoError(t, app.CloseHost())
	require.NoError(t, app.Run(ctx))

	app.handleDisconnectEvent(evtNotConnected(newHost(t).ID()))
	time.Sleep(100 * time.Millisecond)
	require.Empty(t, rec.snapshot())
}

// reachableFake returns a reconnect fake listing ids; it dials each target
// only once reachable is set, like a boot sync whose dials fail until then.
func reachableFake(rec *callRecorder, reachable *atomic.Bool, ids []peer.ID, targets ...host.Host) reconnectFunc {
	return func(ctx context.Context, h host.Host) ([]peer.ID, error) {
		rec.add()
		if reachable.Load() {
			for _, t := range targets {
				_ = dial(ctx, h, t)
			}
		}
		return ids, nil
	}
}

// A peer listed by the startup sync whose dial failed is retried until it is
// reachable, without any disconnect event (#2915).
func TestStartupDialFailureRetriedUntilConnected(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h1, h2 := newHost(t), newHost(t)
	listed := []peer.ID{h1.ID(), h2.ID()} // the boot list includes self once registered
	var reachable atomic.Bool
	rec := &callRecorder{}
	app := New(bus.New(10), h1)
	app.reconnect = reachableFake(rec, &reachable, listed, h2)
	app.backoff = []time.Duration{10 * time.Millisecond}
	app.minSyncInterval = 0

	// startup: sync listed h2 but its dial failed
	app.SeedPending(listed)
	require.NoError(t, app.Run(ctx))

	require.Eventually(t, func() bool { return len(rec.snapshot()) >= 3 }, 2*time.Second, time.Millisecond, "unconnected listed peer not retried")
	require.NotEqual(t, network.Connected, h1.Network().Connectedness(h2.ID()))

	reachable.Store(true)
	require.Eventually(t, func() bool { return h1.Network().Connectedness(h2.ID()) == network.Connected },
		2*time.Second, 5*time.Millisecond, "peer not connected once reachable")

	// connected; self is never pending, so the worker goes idle
	require.Eventually(t, func() bool { return pendingLen(app) == 0 }, 2*time.Second, 5*time.Millisecond)
	time.Sleep(100 * time.Millisecond) // let an in-flight sync finish
	n := len(rec.snapshot())
	time.Sleep(200 * time.Millisecond)
	require.Equal(t, n, len(rec.snapshot()), "worker must stop once connected")
}

func pendingLen(a *App) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.pending)
}

// A listed peer connected at startup that drops before Run subscribes to
// events is still retried.
func TestStartupPeerDroppedBeforeRunRetried(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h1, h2 := newHost(t), newHost(t)
	rec := &callRecorder{}
	app := New(bus.New(10), h1)
	app.reconnect = func(ctx context.Context, h host.Host) ([]peer.ID, error) {
		rec.add()
		return []peer.ID{h2.ID()}, dial(ctx, h, h2)
	}
	app.backoff = []time.Duration{10 * time.Millisecond}
	app.minSyncInterval = 0
	require.NoError(t, dial(ctx, h1, h2))

	app.SeedPending([]peer.ID{h2.ID()})
	require.NoError(t, h1.Network().ClosePeer(h2.ID())) // no subscriber yet: event lost
	require.Eventually(t, func() bool { return h1.Network().Connectedness(h2.ID()) == network.NotConnected },
		time.Second, time.Millisecond)
	time.Sleep(100 * time.Millisecond) // let the event bus deliver the disconnect before Run subscribes
	require.NoError(t, app.Run(ctx))

	require.Eventually(t, func() bool { return h1.Network().Connectedness(h2.ID()) == network.Connected },
		2*time.Second, 5*time.Millisecond, "peer dropped before Run not retried")
}

// A peer first listed by a worker sync while the worker is at a long backoff
// is retried after the first backoff step, not the current one.
func TestNewlyListedPeerResetsBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a, b := newHost(t).ID(), newHost(t).ID()
	rec := &callRecorder{}
	app := New(bus.New(10), newHost(t))
	app.reconnect = func(context.Context, host.Host) ([]peer.ID, error) {
		if rec.add() >= 3 {
			return []peer.ID{a, b}, nil // b registers (unreachable) by the 3rd sync
		}
		return []peer.ID{a}, nil
	}
	// the 3rd failed attempt waits 1h unless b being listed resets the backoff
	app.backoff = []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, time.Hour}
	app.minSyncInterval = 0
	go app.reconnectWorker(ctx)

	app.SeedPending([]peer.ID{a})
	require.Eventually(t, func() bool { return len(rec.snapshot()) >= 4 }, time.Second, time.Millisecond, "newly listed peer waits the long backoff")
}

// A sync started by a disconnect also queues other listed peers it failed to
// dial (e.g. the new ID of a restarted node), not only the disconnected one.
func TestWorkerSyncQueuesUnconnectedListedPeer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h1, h2, h3 := newHost(t), newHost(t), newHost(t)
	var h3Reachable atomic.Bool
	rec := &callRecorder{}
	app := New(bus.New(10), h1)
	app.reconnect = func(ctx context.Context, h host.Host) ([]peer.ID, error) {
		rec.add()
		_ = dial(ctx, h, h2)
		if h3Reachable.Load() {
			_ = dial(ctx, h, h3)
		}
		return []peer.ID{h2.ID(), h3.ID()}, nil
	}
	app.backoff = []time.Duration{10 * time.Millisecond}
	app.minSyncInterval = 0
	require.NoError(t, app.Run(ctx))
	require.NoError(t, dial(ctx, h1, h2))

	require.NoError(t, h1.Network().ClosePeer(h2.ID()))
	require.Eventually(t, func() bool { return h1.Network().Connectedness(h2.ID()) == network.Connected },
		2*time.Second, 5*time.Millisecond)
	require.Eventually(t, func() bool { return len(rec.snapshot()) >= 3 }, 2*time.Second, time.Millisecond, "unconnected listed peer not retried")

	h3Reachable.Store(true)
	require.Eventually(t, func() bool { return h1.Network().Connectedness(h3.ID()) == network.Connected },
		2*time.Second, 5*time.Millisecond, "listed peer not connected once reachable")
}

// A seeded peer that connects and then drops is retried right away, not
// after the backoff it was seeded with.
func TestSeededPeerDisconnectSkipsBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := newHost(t).ID()
	rec := &callRecorder{}
	app := New(bus.New(10), newHost(t))
	app.reconnect = rec.knows(a)
	app.backoff = []time.Duration{time.Hour}
	app.minSyncInterval = 0
	go app.reconnectWorker(ctx)

	app.SeedPending([]peer.ID{a})
	require.Eventually(t, func() bool { return len(rec.snapshot()) == 1 }, time.Second, time.Millisecond)

	// a connected (e.g. dialed by the other side) and dropped during the 1h backoff
	app.handleDisconnectEvent(evtNotConnected(a))
	require.Eventually(t, func() bool { return len(rec.snapshot()) == 2 }, time.Second, time.Millisecond)

	// further events for a keep the backoff
	app.handleDisconnectEvent(evtNotConnected(a))
	time.Sleep(50 * time.Millisecond)
	require.Len(t, rec.snapshot(), 2)
}

func TestSeedPendingSkipsSelfAndPrunesConnected(t *testing.T) {
	h1, h2, h3 := newHost(t), newHost(t), newHost(t)
	app := New(bus.New(10), h1)
	require.NoError(t, dial(context.Background(), h1, h2))

	app.SeedPending([]peer.ID{h1.ID(), h2.ID(), h3.ID(), h3.ID()})
	require.Equal(t, map[peer.ID]pendingPeer{h2.ID(): {seeded: true}, h3.ID(): {seeded: true}}, app.pending)

	n, _ := app.prunePending(nil, false)
	require.Equal(t, 1, n, "connected peer must be dropped")
	require.Contains(t, app.pending, h3.ID())

	// a worker sync queues only unconnected listed peers
	_, seeded := app.prunePending([]peer.ID{h1.ID(), h2.ID(), h3.ID()}, true)
	require.Zero(t, seeded)
}
