package setup

import (
	"context"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/metrics"
	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// counter reads a counter series from the registry /metrics is served from.
func counter(t *testing.T, name, peerType string) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == metricnames.PeerTypeLabel && label.GetValue() == peerType {
					return metric.GetCounter().GetValue()
				}
			}
		}
	}
	t.Fatalf("%s{%s=%q} not registered", name, metricnames.PeerTypeLabel, peerType)
	return 0
}

// eventually polls here, not in require.Eventually: counter can't fail a test off
// its goroutine.
func eventually(t *testing.T, name, peerType string, want float64) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for counter(t, name, peerType) != want {
		if time.Now().After(deadline) {
			t.Fatalf("%s{%s}: got %v, want %v", name, peerType, counter(t, name, peerType), want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

type fakeConn struct {
	network.Conn
	remote peer.ID
}

func (c fakeConn) RemotePeer() peer.ID { return c.remote }

func alreadyIdentified(network.Conn) <-chan struct{} {
	done := make(chan struct{})
	close(done)
	return done
}

func classifyAs(peerType string) func(peer.ID) string {
	return func(peer.ID) string { return peerType }
}

func TestPeerMetricsCountPeersNotConnections(t *testing.T) {
	const peerType = metrics.P2PPeerTypeOracle
	notifiee := newPeerMetricsNotifiee(classifyAs(peerType), alreadyIdentified)
	a, b := fakeConn{remote: "peer-a"}, fakeConn{remote: "peer-b"}

	connects := counter(t, metricnames.P2PPeerConnects, peerType)
	disconnects := counter(t, metricnames.P2PPeerDisconnects, peerType)
	check := func(wantConnects, wantDisconnects float64) {
		t.Helper()
		eventually(t, metricnames.P2PPeerConnects, peerType, connects+wantConnects)
		eventually(t, metricnames.P2PPeerDisconnects, peerType, disconnects+wantDisconnects)
	}

	notifiee.Connected(nil, a)
	check(1, 0) // first connection
	notifiee.Connected(nil, a)
	check(1, 0) // second connection to the same peer
	notifiee.Connected(nil, b)
	check(2, 0) // another peer

	notifiee.Disconnected(nil, a)
	check(2, 0) // spare connection closed
	notifiee.Disconnected(nil, a)
	check(2, 1) // last connection closed
	notifiee.Disconnected(nil, b)
	check(2, 2) // other peer closed

	notifiee.Connected(nil, a)
	check(3, 2) // reconnect
}

func TestPeerMetricsConnectionOpenedBeforeRegistering(t *testing.T) {
	const peerType = metrics.P2PPeerTypeOracle
	notifiee := newPeerMetricsNotifiee(classifyAs(peerType), alreadyIdentified)
	early := fakeConn{remote: "peer-early"}

	connects := counter(t, metricnames.P2PPeerConnects, peerType)
	disconnects := counter(t, metricnames.P2PPeerDisconnects, peerType)

	notifiee.Disconnected(nil, early)
	assert.Equal(t, disconnects+1, counter(t, metricnames.P2PPeerDisconnects, peerType))

	// the peer stays visible to both counters
	notifiee.Connected(nil, early)
	eventually(t, metricnames.P2PPeerConnects, peerType, connects+1)
	notifiee.Disconnected(nil, early)
	assert.Equal(t, disconnects+2, counter(t, metricnames.P2PPeerDisconnects, peerType))
}

func TestPeerMetricsConnectWaitsForIdentify(t *testing.T) {
	// the type is only knowable after identify, so it must be read after it
	const peerType = metrics.P2PPeerTypeBoot
	identified := make(chan struct{})
	current := metrics.P2PPeerTypeUnknown
	classify := func(peer.ID) string { return current }
	notifiee := newPeerMetricsNotifiee(classify, func(network.Conn) <-chan struct{} { return identified })

	connects := counter(t, metricnames.P2PPeerConnects, peerType)
	unknown := counter(t, metricnames.P2PPeerConnects, metrics.P2PPeerTypeUnknown)

	notifiee.Connected(nil, fakeConn{remote: "peer-slow"})
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, connects, counter(t, metricnames.P2PPeerConnects, peerType))

	current = peerType // the channel close below orders this before the read
	close(identified)
	eventually(t, metricnames.P2PPeerConnects, peerType, connects+1)
	assert.Equal(t, unknown, counter(t, metricnames.P2PPeerConnects, metrics.P2PPeerTypeUnknown))
}

func TestRegisterPeerMetricsOnHost(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	h, err := NewHost(ctx)
	require.NoError(t, err)
	defer h.Close()

	// built like the boot API (host only) and like a node (host and pubsub)
	boot, err := NewHost(ctx)
	require.NoError(t, err)
	defer boot.Close()
	oracle, err := NewHost(ctx)
	require.NoError(t, err)
	defer oracle.Close()
	_, err = MakePubsub(ctx, oracle)
	require.NoError(t, err)

	RegisterPeerMetrics(h)

	for _, tc := range []struct {
		name     string
		peerType string
		remote   peer.AddrInfo
	}{
		{"boot API", metrics.P2PPeerTypeBoot, peer.AddrInfo{ID: boot.ID(), Addrs: boot.Addrs()}},
		{"oracle", metrics.P2PPeerTypeOracle, peer.AddrInfo{ID: oracle.ID(), Addrs: oracle.Addrs()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			connects := counter(t, metricnames.P2PPeerConnects, tc.peerType)
			disconnects := counter(t, metricnames.P2PPeerDisconnects, tc.peerType)

			require.NoError(t, h.Connect(ctx, tc.remote))
			eventually(t, metricnames.P2PPeerConnects, tc.peerType, connects+1)

			require.NoError(t, h.Network().ClosePeer(tc.remote.ID))
			eventually(t, metricnames.P2PPeerDisconnects, tc.peerType, disconnects+1)

			require.NoError(t, h.Connect(ctx, tc.remote))
			eventually(t, metricnames.P2PPeerConnects, tc.peerType, connects+2)
		})
	}
}
