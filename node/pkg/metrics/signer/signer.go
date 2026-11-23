// Package signer counts failed signer key rotations (updateOracle), so a
// rotation that cannot land alerts long before the old key expires.
//
// Like pkg/metrics/submission it registers only on Register (called at signer
// startup) or first use, so binaries that only import the chain helper (e.g.
// sentinel) export none of these series.
package signer

import (
	"sync"

	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	"github.com/prometheus/client_golang/prometheus"
)

// Rotation failure reasons.
const (
	// updateOracle was not broadcast (key setup, tx build, or delegator and
	// direct fallback failed before send) or the node refused it (e.g.
	// insufficient funds: old key unfunded): the old key keeps signing
	ReasonNotSent = "not_sent"
	// updateOracle was (or may have been) sent, but the new key was not seen
	// whitelisted on-chain within the verify budget
	ReasonNotLanded = "not_landed"
)

var (
	registerOnce     sync.Once
	rotationFailures = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: metricnames.SignerRotationFailures,
		Help: "Signer key rotation attempts that failed, by reason (not_sent: not broadcast or refused by the node; not_landed: not seen on-chain); retried with exponential backoff",
	}, []string{metricnames.ReasonLabel})
)

// Register adds the collector to the default registry with every series at 0,
// so the first failure shows up in rate() and increase(). Idempotent.
func Register() {
	registerOnce.Do(func() {
		prometheus.MustRegister(rotationFailures)
		for _, reason := range []string{ReasonNotSent, ReasonNotLanded} {
			rotationFailures.WithLabelValues(reason)
		}
	})
}

// RecordRotationFailure counts one failed rotation attempt.
func RecordRotationFailure(reason string) {
	Register()
	rotationFailures.WithLabelValues(reason).Inc()
}
