package balance

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	balanceGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "orakl_balance_kaia",
		Help: "Latest native balance (KAIA) of a watched wallet.",
	}, []string{"address", "role"})
	minimumGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "orakl_balance_minimum_kaia",
		Help: "Configured minimum balance (KAIA) of a watched wallet (BALANCE_WALLETS).",
	}, []string{"address", "role"})
	lastSuccessGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "orakl_balance_last_success_timestamp_seconds",
		Help: "Unix time of the last successful balance read of a watched wallet.",
	}, []string{"address", "role"})
	rpcErrorsCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "orakl_balance_rpc_errors_total",
		Help: "Failed balance reads.",
	})
)

func init() {
	prometheus.MustRegister(balanceGauge, minimumGauge, lastSuccessGauge, rpcErrorsCounter)
}

func registerWalletMetrics(ws []Wallet) {
	for _, w := range ws {
		minimumGauge.WithLabelValues(w.Address.Hex(), w.Tag).Set(w.Minimum)
	}
}

func recordBalance(w Wallet, balance float64, at time.Time) {
	balanceGauge.WithLabelValues(w.Address.Hex(), w.Tag).Set(balance)
	lastSuccessGauge.WithLabelValues(w.Address.Hex(), w.Tag).Set(float64(at.Unix()))
}
