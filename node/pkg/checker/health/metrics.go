package health

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	upGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "orakl_health_up",
		Help: "1 if the last health check of a target succeeded, else 0.",
	}, []string{"target"})
	lastCheckGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orakl_health_last_check_timestamp_seconds",
		Help: "Unix time of the last completed health check round.",
	})
)

func init() {
	prometheus.MustRegister(upGauge, lastCheckGauge)
}

func recordHealth(target string, alive bool) {
	v := 0.0
	if alive {
		v = 1
	}
	upGauge.WithLabelValues(target).Set(v)
}

func recordRound(at time.Time) {
	lastCheckGauge.Set(float64(at.Unix()))
}
