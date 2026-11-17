package signer

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	expiryGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "orakl_signer_expiry_timestamp_seconds",
		Help: "Expiration (Unix seconds) of a signer registered in the submission proxy.",
	}, []string{"address"})
	countGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orakl_signer_count",
		Help: "Signers registered in the submission proxy at the last successful read.",
	})
	lastSuccessGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orakl_signer_last_success_timestamp_seconds",
		Help: "Unix time of the last check that read all signer expirations.",
	})
	errorsCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "orakl_signer_errors_total",
		Help: "Failed signer address or expiration reads.",
	})
)

func init() {
	prometheus.MustRegister(expiryGauge, countGauge, lastSuccessGauge, errorsCounter)
}

func recordLastSuccess(at time.Time) {
	lastSuccessGauge.Set(float64(at.Unix()))
}
