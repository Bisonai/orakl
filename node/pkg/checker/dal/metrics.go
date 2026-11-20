package dal

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	restAggregateTimestampGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "orakl_dal_check_rest_aggregate_timestamp_seconds",
		Help: "aggregateTime (Unix seconds) of the latest DAL REST /latest-data-feeds/all entry per symbol.",
	}, []string{"symbol"})
	restDurationGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orakl_dal_check_rest_duration_seconds",
		Help: "Duration of the last DAL REST /latest-data-feeds/all request.",
	})
	restLastSuccessGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orakl_dal_check_rest_last_success_timestamp_seconds",
		Help: "Unix time of the last successful DAL REST read.",
	})
	restErrorsCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "orakl_dal_check_rest_errors_total",
		Help: "Failed DAL REST reads.",
	})
	restEmptyFeedsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orakl_dal_check_rest_empty_feeds",
		Help: "Entries with an empty field in the last DAL REST response.",
	})
	restMissingFeedsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orakl_dal_check_rest_missing_feeds",
		Help: "Configured symbols missing from the last DAL REST response.",
	})
	wsLastPushGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "orakl_dal_check_ws_last_push_timestamp_seconds",
		Help: "Unix time the sentinel last received a DAL WebSocket push per symbol.",
	}, []string{"symbol"})
)

func init() {
	prometheus.MustRegister(
		restAggregateTimestampGauge,
		restDurationGauge,
		restLastSuccessGauge,
		restErrorsCounter,
		restEmptyFeedsGauge,
		restMissingFeedsGauge,
		wsLastPushGauge,
	)
}
