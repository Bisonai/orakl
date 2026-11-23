package dalstream

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Values of the `reason` label of dal_stream_rejected_connections_total.
const (
	RejectMaxConnections = "max_connections"
	RejectMaxPerIP       = "max_per_ip"
)

var (
	activeConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "dal_stream_active_connections",
		Help: "Current number of open SSE streams",
	})
	connectionsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dal_stream_connections_total",
		Help: "Total number of accepted SSE streams",
	})
	rejectedConnections = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dal_stream_rejected_connections_total",
		Help: "SSE streams rejected because a connection limit was hit",
	}, []string{"reason"})
	slowClientDrops = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dal_stream_slow_client_drops_total",
		Help: "SSE streams closed because the client did not keep up",
	})
	bytesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dal_stream_bytes_sent_total",
		Help: "Bytes written to SSE streams",
	})
	upstreamConnected = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "dal_stream_upstream_connected",
		Help: "1 while the DAL WebSocket is connected",
	})
	upstreamReconnects = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dal_stream_upstream_reconnects_total",
		Help: "DAL WebSocket connection attempts after the first",
	})
	changedPerTick = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "dal_stream_changed_symbols_per_tick",
		Help:    "Symbols whose value changed in a tick",
		Buckets: []float64{0, 1, 5, 10, 25, 50, 100, 150, 250, 500},
	})
)

// registerUpstreamAge exports seconds since the last upstream message of h;
// call the returned func to unregister.
func registerUpstreamAge(reg prometheus.Registerer, h *Hub) (func(), error) {
	g := prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "dal_stream_upstream_last_message_age_seconds",
		Help: "Seconds since the last DAL message; -1 before the first one",
	}, h.lastMessageAge)
	if err := reg.Register(g); err != nil {
		return nil, err
	}
	return func() { reg.Unregister(g) }, nil
}
