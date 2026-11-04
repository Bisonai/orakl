package metrics

// Counters of p2p messages published by this node, by message type. Bytes are
// encoded payload only: no signature, framing or packet overhead, no per-peer
// fan-out, and no messages relayed for other nodes.

import (
	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	p2pPublishedMessages = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: metricnames.P2PPublishedMessages,
		Help: "P2P messages successfully published by this node, by message type",
	}, []string{metricnames.TypeLabel})
	p2pPublishedBytes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: metricnames.P2PPublishedBytes,
		Help: "Encoded payload bytes of P2P messages successfully published by this node, by message type",
	}, []string{metricnames.TypeLabel})
)

// RecordP2PPublished counts one successfully published message of msgType with
// an encoded payload of size bytes.
func RecordP2PPublished(msgType string, size int) {
	p2pPublishedMessages.WithLabelValues(msgType).Inc()
	p2pPublishedBytes.WithLabelValues(msgType).Add(float64(size))
}
