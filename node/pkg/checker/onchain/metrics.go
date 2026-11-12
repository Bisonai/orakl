package onchain

import "github.com/prometheus/client_golang/prometheus"

var (
	// Raw on-chain answer (no decimals applied), same scale as
	// orakl_node_global_aggregate_value so the two can be compared directly.
	answerGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "orakl_onchain_answer",
		Help: "Latest on-chain feed answer (raw integer, decimals not applied).",
	}, []string{"name"})
	updatedGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "orakl_onchain_updated_timestamp_seconds",
		Help: "Block timestamp of the latest on-chain feed update.",
	}, []string{"name"})
	updatesCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "orakl_onchain_updates_total",
		Help: "FeedUpdated events seen since start.",
	}, []string{"name"})
	expectedIntervalGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "orakl_onchain_expected_interval_seconds",
		Help: "Configured submit interval of the feed (orakl-config submitInterval).",
	}, []string{"name"})
	feedsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orakl_onchain_feeds",
		Help: "Number of feeds tracked by the on-chain reader.",
	})
	lastBlockGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orakl_onchain_last_block",
		Help: "Last block number processed by the on-chain reader.",
	})
	lastPollSuccessGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orakl_onchain_last_poll_success_timestamp_seconds",
		Help: "Unix time of the last successful poll.",
	})
	rpcErrorsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "orakl_onchain_rpc_errors_total",
		Help: "RPC errors by method.",
	}, []string{"method"})
)

func init() {
	prometheus.MustRegister(answerGauge, updatedGauge, updatesCounter, expectedIntervalGauge,
		feedsGauge, lastBlockGauge, lastPollSuccessGauge, rpcErrorsCounter)
}

func deleteFeedMetrics(name string) {
	answerGauge.DeleteLabelValues(name)
	updatedGauge.DeleteLabelValues(name)
	updatesCounter.DeleteLabelValues(name)
	expectedIntervalGauge.DeleteLabelValues(name)
}
