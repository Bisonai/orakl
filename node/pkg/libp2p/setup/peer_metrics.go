package setup

import (
	"sync"

	"bisonai.com/miko/node/pkg/metrics"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// RegisterPeerMetrics counts peer connects and disconnects on h, for every peer
// including the boot API. Call it once per host, right after creating it, as the
// host accepts connections from then on.
func RegisterPeerMetrics(h host.Host) {
	h.Network().Notify(newPeerMetricsNotifiee())
}

// newPeerMetricsNotifiee counts per peer, not per connection: a peer can hold
// several (simultaneous dials), and closing a spare one isn't a disconnect.
func newPeerMetricsNotifiee() *network.NotifyBundle {
	var mu sync.Mutex
	conns := make(map[peer.ID]int)

	return &network.NotifyBundle{
		ConnectedF: func(_ network.Network, c network.Conn) {
			mu.Lock()
			defer mu.Unlock()
			conns[c.RemotePeer()]++
			if conns[c.RemotePeer()] == 1 {
				metrics.RecordP2PPeerConnect()
			}
		},
		DisconnectedF: func(_ network.Network, c network.Conn) {
			mu.Lock()
			defer mu.Unlock()
			// <= 1: a connection opened before registering was never counted
			if conns[c.RemotePeer()] <= 1 {
				delete(conns, c.RemotePeer())
				metrics.RecordP2PPeerDisconnect()
				return
			}
			conns[c.RemotePeer()]--
		},
	}
}
