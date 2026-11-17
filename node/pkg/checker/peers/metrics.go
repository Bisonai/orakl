package peers

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	peerCountGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orakl_peers_count",
		Help: "Peer count of the orakl offchain cluster (max over samples from the node admin API).",
	})
	lastSuccessGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orakl_peers_last_success_timestamp_seconds",
		Help: "Unix time of the last successful peer count read.",
	})
	errorsCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "orakl_peers_errors_total",
		Help: "Failed peer count reads.",
	})
)

func init() {
	prometheus.MustRegister(peerCountGauge, lastSuccessGauge, errorsCounter)
}

func recordPeerCount(count int, at time.Time) {
	peerCountGauge.Set(float64(count))
	lastSuccessGauge.Set(float64(at.Unix()))
}
