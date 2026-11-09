package setup

import (
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
	m := newPeerMetrics(
		func(p peer.ID) string { return peerType(h, p) },
		identifyWait(h),
	)
	h.Network().Notify(&network.NotifyBundle{
		ConnectedF:    func(_ network.Network, c network.Conn) { m.connected(c) },
		DisconnectedF: func(_ network.Network, c network.Conn) { m.disconnected(c) },
	})
}

// peerType tells oracle peers from the boot API host by what they advertise: every
// node runs gossipsub and the boot API host doesn't. Its peer ID can't be used, as
// it's a new random one on each boot API start. Works once identify has completed
// on a connection, and for about a minute past the last one closing, until the
// peerstore forgets the peer.
func peerType(h host.Host, p peer.ID) string {
	store := h.Peerstore()
	if proto, err := store.FirstSupportedProtocol(p, pubsub.GossipSubDefaultProtocols...); err == nil && proto != "" {
		return metrics.P2PPeerTypeOracle
	}
	if protocols, err := store.GetProtocols(p); err != nil || len(protocols) == 0 {
		return metrics.P2PPeerTypeUnknown
	}
	return metrics.P2PPeerTypeBoot
}

// identifyWait returns a function giving a channel that closes when identify has
// finished on a connection, successful or not.
func identifyWait(h host.Host) func(network.Conn) <-chan struct{} {
	if ids, ok := h.(interface{ IDService() identify.IDService }); ok {
		return ids.IDService().IdentifyWait
	}
	// not a host from NewHost: nothing to wait for, the peer stays unknown
	done := make(chan struct{})
	close(done)
	return func(network.Conn) <-chan struct{} { return done }
}

// peerMetrics counts per peer, not per connection: a peer can hold several
// (simultaneous dials), and closing a spare one isn't a disconnect.
//
// Network notifications rather than EvtPeerConnectednessChanged: that event's
// emitter blocks on its slowest subscriber, and the libp2p helper's sleeps minutes
// per disconnect, so counts could lag most when peers flap.
type peerMetrics struct {
	classify   func(peer.ID) string
	identified func(network.Conn) <-chan struct{}

	mu    sync.Mutex
	peers map[peer.ID]*peerConns
}

type peerConns struct {
	conns   int
	counted bool // connect recorded
}

func newPeerMetrics(
	classify func(peer.ID) string,
	identified func(network.Conn) <-chan struct{},
) *peerMetrics {
	return &peerMetrics{
		classify:   classify,
		identified: identified,
		peers:      make(map[peer.ID]*peerConns),
	}
}

func (m *peerMetrics) connected(c network.Conn) {
	p := c.RemotePeer()

	m.mu.Lock()
	state, ok := m.peers[p]
	if !ok {
		state = &peerConns{}
		m.peers[p] = state
	}
	state.conns++
	m.mu.Unlock()

	// the type isn't known yet: Connected fires before identify. Waits on each
	// connection, as the first can close or fail while another one succeeds.
	go func() {
		<-m.identified(c)
		m.identifyDone(p)
	}()
}

// identifyDone records the connect at the first identify that gives the peer's
// type. Until then the connect stays pending, and a peer with no type by its last
// disconnect records it there, so per-type connects and disconnects pair up.
func (m *peerMetrics) identifyDone(p peer.ID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.peers[p]
	if !ok || state.counted {
		return
	}
	peerType := m.classify(p)
	if peerType == metrics.P2PPeerTypeUnknown {
		return
	}
	state.counted = true
	metrics.RecordP2PPeerConnect(peerType)
}

func (m *peerMetrics) disconnected(c network.Conn) {
	p := c.RemotePeer()

	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.peers[p]
	if ok && state.conns > 1 {
		state.conns--
		return
	}
	delete(m.peers, p)

	peerType := m.classify(p)
	// !ok: a connection opened before registering, its connect was never seen
	if ok && !state.counted {
		metrics.RecordP2PPeerConnect(peerType)
	}
	metrics.RecordP2PPeerDisconnect(peerType)
}
