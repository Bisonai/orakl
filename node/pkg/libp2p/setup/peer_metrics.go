package setup

import (
	"slices"
	"sync"

	"bisonai.com/miko/node/pkg/metrics"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
)

// RegisterPeerMetrics counts peer connects and disconnects on h, by peer type.
// Call it once per host, right after creating it, as the host accepts connections
// from then on.
func RegisterPeerMetrics(h host.Host) {
	h.Network().Notify(newPeerMetricsNotifiee(
		func(p peer.ID) string { return peerType(h, p) },
		identifyWait(h),
	))
}

// peerType tells oracle peers from the boot API host by what they advertise: every
// node runs gossipsub and the boot API host doesn't. Its peer ID can't be used, as
// it's a new random one on each boot API start. Works once identify has completed
// on a connection, and for about a minute past the last one closing, until the
// peerstore forgets the peer.
func peerType(h host.Host, p peer.ID) string {
	protocols, err := h.Peerstore().GetProtocols(p)
	if err != nil || len(protocols) == 0 {
		return metrics.P2PPeerTypeUnknown
	}
	for _, protocol := range pubsub.GossipSubDefaultProtocols {
		if slices.Contains(protocols, protocol) {
			return metrics.P2PPeerTypeOracle
		}
	}
	return metrics.P2PPeerTypeBoot
}

// identifyWait returns a function giving a channel that closes when identify has
// finished on a connection, successful or not.
func identifyWait(h host.Host) func(network.Conn) <-chan struct{} {
	if ids, ok := h.(interface{ IDService() identify.IDService }); ok {
		return ids.IDService().IdentifyWait
	}
	// not a host from NewHost: count without waiting, as unknown
	done := make(chan struct{})
	close(done)
	return func(network.Conn) <-chan struct{} { return done }
}

// newPeerMetricsNotifiee counts per peer, not per connection: a peer can hold
// several (simultaneous dials), and closing a spare one isn't a disconnect.
//
// Network notifications rather than EvtPeerConnectednessChanged: that event's
// emitter blocks on its slowest subscriber, and the libp2p helper's sleeps minutes
// per disconnect, so counts could lag most when peers flap.
func newPeerMetricsNotifiee(
	classify func(peer.ID) string,
	identified func(network.Conn) <-chan struct{},
) *network.NotifyBundle {
	var mu sync.Mutex
	conns := make(map[peer.ID]int)

	return &network.NotifyBundle{
		ConnectedF: func(_ network.Network, c network.Conn) {
			mu.Lock()
			defer mu.Unlock()
			conns[c.RemotePeer()]++
			if conns[c.RemotePeer()] == 1 {
				// Connected fires before identify, so the type isn't known yet
				go func() {
					<-identified(c)
					metrics.RecordP2PPeerConnect(classify(c.RemotePeer()))
				}()
			}
		},
		DisconnectedF: func(_ network.Network, c network.Conn) {
			mu.Lock()
			defer mu.Unlock()
			// <= 1: a connection opened before registering was never counted
			if conns[c.RemotePeer()] <= 1 {
				delete(conns, c.RemotePeer())
				metrics.RecordP2PPeerDisconnect(classify(c.RemotePeer()))
				return
			}
			conns[c.RemotePeer()]--
		},
	}
}
