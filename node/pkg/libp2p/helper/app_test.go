package helper

import (
	"context"
	"errors"
	"sync"
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
