package setup

import (
	"context"
	"testing"
	"time"

	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// counter reads a counter from the registry /metrics is served from.
func counter(t *testing.T, name string) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() == name {
			return family.GetMetric()[0].GetCounter().GetValue()
		}
	}
	t.Fatalf("%s not registered", name)
	return 0
}

type fakeConn struct {
	network.Conn
	remote peer.ID
}

func (c fakeConn) RemotePeer() peer.ID { return c.remote }

func TestPeerMetricsCountPeersNotConnections(t *testing.T) {
	notifiee := newPeerMetricsNotifiee()
	a, b := fakeConn{remote: "peer-a"}, fakeConn{remote: "peer-b"}

	connects := counter(t, metricnames.P2PPeerConnects)
	disconnects := counter(t, metricnames.P2PPeerDisconnects)
	check := func(wantConnects, wantDisconnects float64, msg string) {
		t.Helper()
		assert.Equal(t, connects+wantConnects, counter(t, metricnames.P2PPeerConnects), msg)
		assert.Equal(t, disconnects+wantDisconnects, counter(t, metricnames.P2PPeerDisconnects), msg)
	}

	notifiee.Connected(nil, a)
	check(1, 0, "first connection")
	notifiee.Connected(nil, a)
	check(1, 0, "second connection to the same peer")
	notifiee.Connected(nil, b)
	check(2, 0, "another peer")

	notifiee.Disconnected(nil, a)
	check(2, 0, "spare connection closed")
	notifiee.Disconnected(nil, a)
	check(2, 1, "last connection closed")
	notifiee.Disconnected(nil, b)
	check(2, 2, "other peer closed")

	notifiee.Connected(nil, a)
	check(3, 2, "reconnect")
}

func TestPeerMetricsConnectionOpenedBeforeRegistering(t *testing.T) {
	notifiee := newPeerMetricsNotifiee()
	early := fakeConn{remote: "peer-early"}

	connects := counter(t, metricnames.P2PPeerConnects)
	disconnects := counter(t, metricnames.P2PPeerDisconnects)

	notifiee.Disconnected(nil, early)
	assert.Equal(t, disconnects+1, counter(t, metricnames.P2PPeerDisconnects))

	// the peer stays visible to both counters
	notifiee.Connected(nil, early)
	assert.Equal(t, connects+1, counter(t, metricnames.P2PPeerConnects))
	notifiee.Disconnected(nil, early)
	assert.Equal(t, disconnects+2, counter(t, metricnames.P2PPeerDisconnects))
}

func TestRegisterPeerMetricsOnHost(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	h, err := NewHost(ctx)
	require.NoError(t, err)
	defer h.Close()
	other, err := NewHost(ctx)
	require.NoError(t, err)
	defer other.Close()

	RegisterPeerMetrics(h)

	connects := counter(t, metricnames.P2PPeerConnects)
	disconnects := counter(t, metricnames.P2PPeerDisconnects)
	eventually := func(name string, want float64) {
		t.Helper()
		require.Eventually(t, func() bool { return counter(t, name) == want }, 5*time.Second, 10*time.Millisecond, name)
	}

	info := peer.AddrInfo{ID: other.ID(), Addrs: other.Addrs()}
	require.NoError(t, h.Connect(ctx, info))
	eventually(metricnames.P2PPeerConnects, connects+1)

	require.NoError(t, h.Network().ClosePeer(other.ID()))
	eventually(metricnames.P2PPeerDisconnects, disconnects+1)

	require.NoError(t, h.Connect(ctx, info))
	eventually(metricnames.P2PPeerConnects, connects+2)
}
