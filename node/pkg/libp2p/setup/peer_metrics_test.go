package setup

import (
	"context"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/metrics"
	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
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

type fakeConn struct {
	network.Conn
	remote peer.ID
}

func (c fakeConn) RemotePeer() peer.ID { return c.remote }

// newManualPeerMetrics never finishes identify by itself: tests call identifyDone,
// so each count can be checked right away, including that nothing changed. The
// goroutines waiting on the nil channel live until the test binary exits.
func newManualPeerMetrics(classify func(peer.ID) string) *peerMetrics {
	return newPeerMetrics(classify, func(network.Conn) <-chan struct{} { return nil })
}

func classifyAs(peerType string) func(peer.ID) string {
	return func(peer.ID) string { return peerType }
}

// peerCounts checks changes in the counters of one peer type since it was created.
func peerCounts(t *testing.T, peerType string) func(wantConnects, wantDisconnects float64, msg string) {
	t.Helper()
	connects := counter(t, metricnames.P2PPeerConnects, peerType)
	disconnects := counter(t, metricnames.P2PPeerDisconnects, peerType)
	return func(wantConnects, wantDisconnects float64, msg string) {
		t.Helper()
		assert.Equal(t, connects+wantConnects, counter(t, metricnames.P2PPeerConnects, peerType), msg)
		assert.Equal(t, disconnects+wantDisconnects, counter(t, metricnames.P2PPeerDisconnects, peerType), msg)
	}
}

func TestPeerMetricsCountPeersNotConnections(t *testing.T) {
	m := newManualPeerMetrics(classifyAs(metrics.P2PPeerTypeOracle))
	a, b := fakeConn{remote: "peer-a"}, fakeConn{remote: "peer-b"}
	check := peerCounts(t, metrics.P2PPeerTypeOracle)

	m.connected(a)
	check(0, 0, "before identify")
	m.identifyDone(a.remote)
	check(1, 0, "first connection")
	m.connected(a)
	m.identifyDone(a.remote)
	check(1, 0, "second connection to the same peer")
	m.connected(b)
	m.identifyDone(b.remote)
	check(2, 0, "another peer")

	m.disconnected(a)
	check(2, 0, "spare connection closed")
	m.disconnected(a)
	check(2, 1, "last connection closed")
	m.disconnected(b)
	check(2, 2, "other peer closed")

	m.identifyDone(a.remote)
	check(2, 2, "identify ending after the peer left")

	m.connected(a)
	m.identifyDone(a.remote)
	check(3, 2, "reconnect")
	m.disconnected(a)
}

func TestPeerMetricsConnectionOpenedBeforeRegistering(t *testing.T) {
	m := newManualPeerMetrics(classifyAs(metrics.P2PPeerTypeOracle))
	early := fakeConn{remote: "peer-early"}
	check := peerCounts(t, metrics.P2PPeerTypeOracle)

	m.disconnected(early)
	check(0, 1, "no connect was seen")

	// the peer stays visible to both counters
	m.connected(early)
	m.identifyDone(early.remote)
	check(1, 1, "connect")
	m.disconnected(early)
	check(1, 2, "disconnect")
}

func TestPeerMetricsFailedIdentifyDoesNotFixTheType(t *testing.T) {
	current := metrics.P2PPeerTypeUnknown
	m := newManualPeerMetrics(func(peer.ID) string { return current })
	p := fakeConn{remote: "peer-spare"}
	oracle := peerCounts(t, metrics.P2PPeerTypeOracle)
	unknown := peerCounts(t, metrics.P2PPeerTypeUnknown)

	m.connected(p)
	m.connected(p)
	m.identifyDone(p.remote) // the first connection's identify fails
	m.disconnected(p)
	oracle(0, 0, "first identify failed")
	unknown(0, 0, "not unknown while another connection can identify")

	current = metrics.P2PPeerTypeOracle
	m.identifyDone(p.remote)
	oracle(1, 0, "second identify")

	m.disconnected(p)
	oracle(1, 1, "pairs up")
	unknown(0, 0, "never counted as unknown")
}

func TestPeerMetricsPeerGoneBeforeIdentify(t *testing.T) {
	m := newManualPeerMetrics(classifyAs(metrics.P2PPeerTypeUnknown))
	p := fakeConn{remote: "peer-gone"}
	check := peerCounts(t, metrics.P2PPeerTypeUnknown)

	m.connected(p)
	check(0, 0, "pending")
	m.disconnected(p)
	check(1, 1, "connect and disconnect pair up")

	m.identifyDone(p.remote)
	check(1, 1, "identify ending afterwards")
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

	// identify's snapshot of oracle's protocols updates asynchronously: wait until
	// a peer sees gossipsub, or the dial below could race it
	probe, err := NewHost(ctx)
	require.NoError(t, err)
	defer probe.Close()
	require.NoError(t, probe.Connect(ctx, peer.AddrInfo{ID: oracle.ID(), Addrs: oracle.Addrs()}))
	require.Eventually(t, func() bool {
		proto, err := probe.Peerstore().FirstSupportedProtocol(oracle.ID(), pubsub.GossipSubDefaultProtocols...)
		return err == nil && proto != ""
	}, 5*time.Second, 10*time.Millisecond)

	RegisterPeerMetrics(h)

	// polls here, not in require.Eventually: counter can't fail a test off its goroutine
	eventually := func(t *testing.T, name, peerType string, want float64) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for counter(t, name, peerType) != want {
			if time.Now().After(deadline) {
				t.Fatalf("%s{%s}: got %v, want %v", name, peerType, counter(t, name, peerType), want)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

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
