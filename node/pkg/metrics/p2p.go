package metrics

// Counters of p2p messages published by this node, by message type. Bytes are
// encoded payload only: no signature, framing or packet overhead, no per-peer
// fan-out, and no messages relayed for other nodes. A message counts once
// pubsub accepts it for publishing; it may still be dropped or have no peers.

import (
	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	// P2PPeerTypeOracle is a peer that advertises gossipsub, as every node does.
	P2PPeerTypeOracle = "oracle"
	// P2PPeerTypeBoot is an identified peer without gossipsub: the boot API host
	// is the only such host the repo runs.
	P2PPeerTypeBoot = "boot"
	// P2PPeerTypeUnknown is a peer gone without identify ever completing.
	P2PPeerTypeUnknown = "unknown"
)

var (
	p2pPublishedMessages = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: metricnames.P2PPublishedMessages,
		Help: "P2P messages accepted for publishing by this node, by message type",
	}, []string{metricnames.TypeLabel})
	p2pPublishedBytes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: metricnames.P2PPublishedBytes,
		Help: "Encoded payload bytes of P2P messages accepted for publishing by this node, by message type",
	}, []string{metricnames.TypeLabel})

	p2pPeerConnects = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: metricnames.P2PPeerConnects,
		Help: "P2P peers that went from no connection to connected, by peer type",
	}, []string{metricnames.PeerTypeLabel})
	p2pPeerDisconnects = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: metricnames.P2PPeerDisconnects,
		Help: "P2P peers whose last connection closed, by peer type",
	}, []string{metricnames.PeerTypeLabel})
)

func init() {
	// exported at 0, so the first event shows up in rate() and increase()
	for _, peerType := range []string{P2PPeerTypeOracle, P2PPeerTypeBoot, P2PPeerTypeUnknown} {
		p2pPeerConnects.WithLabelValues(peerType)
		p2pPeerDisconnects.WithLabelValues(peerType)
	}
}

// RecordP2PPeerConnect counts one peer going from no connection to connected.
func RecordP2PPeerConnect(peerType string) {
	p2pPeerConnects.WithLabelValues(peerType).Inc()
}

// RecordP2PPeerDisconnect counts one peer losing its last connection.
func RecordP2PPeerDisconnect(peerType string) {
	p2pPeerDisconnects.WithLabelValues(peerType).Inc()
}

// RecordP2PPublished counts one message of msgType accepted for publishing with
// an encoded payload of size bytes.
func RecordP2PPublished(msgType string, size int) {
	p2pPublishedMessages.WithLabelValues(msgType).Inc()
	p2pPublishedBytes.WithLabelValues(msgType).Add(float64(size))
}
