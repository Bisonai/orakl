package helper

import (
	"context"
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

func newHost(t *testing.T) host.Host {
	t.Helper()
	h, err := setup.NewHost(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = h.Close() })
	return h
}

// connectedPair returns a running App on h1, connected to h2.
func connectedPair(t *testing.T, ctx context.Context, reconnect func(context.Context, host.Host) error, backoff []time.Duration) (*App, host.Host, host.Host) {
	t.Helper()
	h1, h2 := newHost(t), newHost(t)

	app := New(bus.New(10), h1)
	app.reconnect = reconnect
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
	var h2 host.Host
	_, h1, h2 := connectedPair(t, ctx, func(ctx context.Context, h host.Host) error {
		rec.add()
		return dial(ctx, h, h2)
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
	var h2 host.Host
	_, h1, h2 := connectedPair(t, ctx, func(ctx context.Context, h host.Host) error {
		if rec.add() < 4 {
			return nil // sync "succeeds" but peer still down
		}
		return dial(ctx, h, h2)
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

func TestReconnectGivesUpAfterMaxAttempts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rec := &callRecorder{}
	h1, h2 := newHost(t), newHost(t)
	app := New(bus.New(10), h1)
	app.reconnect = func(context.Context, host.Host) error { rec.add(); return nil }
	app.backoff = []time.Duration{time.Millisecond}
	app.maxAttempts = 3
	app.minSyncInterval = 0
	require.NoError(t, app.Run(ctx))
	require.NoError(t, dial(ctx, h1, h2))

	require.NoError(t, h1.Network().ClosePeer(h2.ID()))
	require.Eventually(t, func() bool { return len(rec.snapshot()) >= 3 }, 2*time.Second, 5*time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	require.Len(t, rec.snapshot(), 3)
}

func TestDisconnectEventDoesNotBlockEventLoop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	release := make(chan struct{})
	rec := &callRecorder{}
	app := New(bus.New(10), newHost(t))
	app.reconnect = func(ctx context.Context, _ host.Host) error {
		rec.add()
		select {
		case <-release:
		case <-ctx.Done():
		}
		return nil
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
	app.reconnect = func(context.Context, host.Host) error { rec.add(); return nil }
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
	app.reconnect = func(context.Context, host.Host) error { rec.add(); return nil }
	app.backoff = []time.Duration{time.Hour}
	app.minSyncInterval = 0
	stopped := make(chan struct{})
	go func() { app.reconnectWorker(ctx); close(stopped) }()

	app.handleDisconnectEvent(evtNotConnected(newHost(t).ID()))
	require.Eventually(t, func() bool { return len(rec.snapshot()) == 1 }, time.Second, time.Millisecond)

	cancel()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop on ctx cancel")
	}
}

func TestMinSyncIntervalAcrossEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rec := &callRecorder{}
	app := New(bus.New(10), newHost(t))
	app.reconnect = func(context.Context, host.Host) error { rec.add(); return nil }
	app.maxAttempts = 1
	app.minSyncInterval = 150 * time.Millisecond
	go app.reconnectWorker(ctx)

	other := newHost(t).ID()
	app.handleDisconnectEvent(evtNotConnected(other))
	require.Eventually(t, func() bool { return len(rec.snapshot()) == 1 }, time.Second, time.Millisecond)
	time.Sleep(20 * time.Millisecond) // let worker give up and go idle
	app.handleDisconnectEvent(evtNotConnected(other))
	require.Eventually(t, func() bool { return len(rec.snapshot()) == 2 }, time.Second, time.Millisecond)

	times := rec.snapshot()
	require.GreaterOrEqual(t, times[1].Sub(times[0]), app.minSyncInterval)
}

func evtNotConnected(p peer.ID) event.EvtPeerConnectednessChanged {
	return event.EvtPeerConnectednessChanged{Peer: p, Connectedness: network.NotConnected}
}

func evtConnected(p peer.ID) event.EvtPeerConnectednessChanged {
	return event.EvtPeerConnectednessChanged{Peer: p, Connectedness: network.Connected}
}
