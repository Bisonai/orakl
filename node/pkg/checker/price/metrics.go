package price

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	dalPriceGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "orakl_price_check_dal_price",
		Help: "Latest DAL (mainnet) price of a tracked pair.",
	}, []string{"pair"})
	binancePriceGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "orakl_price_check_binance_price",
		Help: "Latest Binance price of a tracked pair.",
	}, []string{"pair"})
	diffRatioGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "orakl_price_check_diff_ratio",
		Help: "|dal - binance| / dal of a tracked pair.",
	}, []string{"pair"})
	thresholdGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orakl_price_check_threshold_ratio",
		Help: "Configured diff threshold (PRICE_DIFF_THRESHOLD).",
	})
	lastSuccessGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orakl_price_check_last_success_timestamp_seconds",
		Help: "Unix time of the last round that fetched both DAL and Binance prices.",
	})
	errorsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "orakl_price_check_errors_total",
		Help: "Failed price fetches per source.",
	}, []string{"source"})
)

func init() {
	prometheus.MustRegister(dalPriceGauge, binancePriceGauge, diffRatioGauge, thresholdGauge, lastSuccessGauge, errorsCounter)
}

func recordPrices(compared map[baseAndQuote][2]float64, at time.Time) {
	dalPriceGauge.Reset()
	binancePriceGauge.Reset()
	diffRatioGauge.Reset()
	for e, p := range compared {
		pair := e.base + "-" + e.quote
		dalPriceGauge.WithLabelValues(pair).Set(p[0])
		binancePriceGauge.WithLabelValues(pair).Set(p[1])
		diffRatioGauge.WithLabelValues(pair).Set(diffRatio(p[0], p[1]))
	}
	lastSuccessGauge.Set(float64(at.Unix()))
}
