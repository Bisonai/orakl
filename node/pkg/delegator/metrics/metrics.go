// Package metrics holds the delegator's Prometheus metrics and serves them on
// a listener separate from the public API (METRICS_ADDR).
package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Results of a sign request (orakl_delegator_sign_requests_total{result}).
const (
	ResultOK              = "ok"
	ResultNotAllowlisted  = "not_allowlisted"
	ResultAuthFailed      = "auth_failed"
	ResultInvalidTx       = "invalid_tx"
	ResultError           = "error"
	StoreOpTokenLookup    = "token_lookup"
	StoreOpContractLookup = "contract_lookup"
	StoreOpFunctionLookup = "function_lookup"
	StoreOpRecordTx       = "record_tx"
	StoreOpAllowlistCount = "allowlist_count"
	RPCMethodDial         = "dial"
	RPCMethodBalanceAt    = "balance_at"
)

var (
	results  = []string{ResultOK, ResultNotAllowlisted, ResultAuthFailed, ResultInvalidTx, ResultError}
	storeOps = []string{StoreOpTokenLookup, StoreOpContractLookup, StoreOpFunctionLookup, StoreOpRecordTx, StoreOpAllowlistCount}
)

var (
	// caller is the name of the API token (SQLite mode; operator-created, so
	// bounded), "" when unauthenticated or in Postgres mode.
	signRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "orakl_delegator_sign_requests_total",
		Help: "POST /sign/v2 requests by result and caller (API token name).",
	}, []string{"result", "caller"})
	signDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "orakl_delegator_sign_duration_seconds",
		Help:    "Time from request to signed response, successful requests only.",
		Buckets: []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
	})
	storeErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "orakl_delegator_store_errors_total",
		Help: "Store (SQLite or Postgres) read/write failures by operation.",
	}, []string{"op"})
	rpcErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "orakl_delegator_rpc_errors_total",
		Help: "RPC errors of the fee payer balance poller by method.",
	}, []string{"method"})

	// registered on first set, so no 0 is exported before the first value
	// (an empty account or allowlist, or a sign in 1970, to an alert)
	lastSignSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orakl_delegator_last_sign_success_timestamp_seconds",
		Help: "Unix time of the last successfully signed transaction.",
	})
	allowlistEntries = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orakl_delegator_allowlist_entries",
		Help: "Allowed (contract, function) entries in the store.",
	})
	feePayerBalance = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orakl_delegator_fee_payer_balance",
		Help: "Fee payer balance in KAIA.",
	})
	feePayerBalanceUpdated = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orakl_delegator_fee_payer_balance_last_update_timestamp_seconds",
		Help: "Unix time of the last successful fee payer balance poll.",
	})
	registerLastSignOnce  sync.Once
	registerAllowlistOnce sync.Once
	registerBalanceOnce   sync.Once
)

func init() {
	prometheus.MustRegister(signRequests, signDuration, storeErrors, rpcErrors)
	// zero series so rate() and absence alerts work before the first event
	for _, r := range results {
		signRequests.WithLabelValues(r, "")
	}
	for _, op := range storeOps {
		storeErrors.WithLabelValues(op)
	}
	for _, m := range []string{RPCMethodDial, RPCMethodBalanceAt} {
		rpcErrors.WithLabelValues(m)
	}
}

// ObserveSign records the outcome of one sign request. The duration is only
// observed for signed responses.
func ObserveSign(result, caller string, d time.Duration) {
	signRequests.WithLabelValues(result, caller).Inc()
	if result == ResultOK {
		signDuration.Observe(d.Seconds())
		registerLastSignOnce.Do(func() { prometheus.MustRegister(lastSignSuccess) })
		lastSignSuccess.SetToCurrentTime()
	}
}

// StoreError counts a failed store operation.
func StoreError(op string) {
	storeErrors.WithLabelValues(op).Inc()
}

// SetAllowlistEntries sets the allowlist size.
func SetAllowlistEntries(n int) {
	registerAllowlistOnce.Do(func() { prometheus.MustRegister(allowlistEntries) })
	allowlistEntries.Set(float64(n))
}

func rpcError(method string) {
	rpcErrors.WithLabelValues(method).Inc()
}

func setFeePayerBalance(kaia float64) {
	registerBalanceOnce.Do(func() {
		prometheus.MustRegister(feePayerBalance, feePayerBalanceUpdated)
	})
	feePayerBalance.Set(kaia)
	feePayerBalanceUpdated.SetToCurrentTime()
}
