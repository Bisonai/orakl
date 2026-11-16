// Package submission counts reporter transactions by fee path, and why the
// delegator could not sign. A direct_fallback means the sending key (reporter,
// or old signer on rotation) paid gas instead of the fee payer, so alert on it.
//
// Kept apart from pkg/metrics, and registered on first use, so binaries that
// only import the chain helper (e.g. sentinel) export none of these series.
package submission

import (
	"sync"

	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	"github.com/prometheus/client_golang/prometheus"
)

// Fee paths.
const (
	// signed by the delegator: the fee payer pays gas
	PathDelegated = "delegated"
	// the delegator failed, sent as a direct tx: the sending key pays gas
	PathDirectFallback = "direct_fallback"
	// no delegator configured, sent as a direct tx
	PathDirect = "direct"
)

// Purposes.
const (
	// feed value submissions
	PurposeSubmit = "submit"
	// signer rotation (updateOracle), sent with the old signer key
	PurposeRotate = "rotate"
)

// Delegator error reasons.
const (
	// the request could not be built or sent, or got no response
	ReasonRequest = "request"
	// the delegator answered with a non-200 status
	ReasonHTTPStatus = "http_status"
	// the response was not a usable signed tx
	ReasonInvalidResponse = "invalid_response"
)

var (
	registerOnce sync.Once
	submissions  = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: metricnames.ReporterSubmissions,
		Help: "Reporter transactions sent, by fee path (delegated, direct_fallback, direct) and purpose (submit, rotate)",
	}, []string{metricnames.PathLabel, metricnames.PurposeLabel})
	delegatorErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: metricnames.ReporterDelegatorErrors,
		Help: "Delegator sign requests that failed, by reason (request, http_status, invalid_response) and purpose (submit, rotate)",
	}, []string{metricnames.ReasonLabel, metricnames.PurposeLabel})
)

// register adds the collectors to the default registry with every series at 0,
// so a later fallback shows up in rate() and increase().
func register() {
	registerOnce.Do(func() {
		prometheus.MustRegister(submissions, delegatorErrors)
		for _, purpose := range []string{PurposeSubmit, PurposeRotate} {
			for _, path := range []string{PathDelegated, PathDirectFallback, PathDirect} {
				submissions.WithLabelValues(path, purpose)
			}
			for _, reason := range []string{ReasonRequest, ReasonHTTPStatus, ReasonInvalidResponse} {
				delegatorErrors.WithLabelValues(reason, purpose)
			}
		}
	})
}

// RecordSubmission counts one transaction sent on path for purpose.
func RecordSubmission(path, purpose string) {
	register()
	submissions.WithLabelValues(path, purpose).Inc()
}

// RecordDelegatorError counts one failed delegator sign request.
func RecordDelegatorError(reason, purpose string) {
	register()
	delegatorErrors.WithLabelValues(reason, purpose).Inc()
}
