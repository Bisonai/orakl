package metrics

// Per-config latest local/global aggregate gauges, served by the node admin
// server on /api/v1/metrics. The `name` label is the config name.

import (
	"sync"
	"time"

	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	localAggregateValue = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: metricnames.LocalAggregateValue,
		Help: "Latest local aggregate value per config",
	}, []string{metricnames.NameLabel})
	localAggregateTimestamp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: metricnames.LocalAggregateTimestamp,
		Help: "Unix timestamp of the latest local aggregate per config (time the config was loaded if none yet)",
	}, []string{metricnames.NameLabel})
	globalAggregateValue = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: metricnames.GlobalAggregateValue,
		Help: "Latest global aggregate value per config",
	}, []string{metricnames.NameLabel})
	globalAggregateRound = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: metricnames.GlobalAggregateRound,
		Help: "Round of the latest global aggregate per config",
	}, []string{metricnames.NameLabel})
	globalAggregateTimestamp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: metricnames.GlobalAggregateTimestamp,
		Help: "Unix timestamp of the latest global aggregate per config (time the config was loaded if none yet)",
	}, []string{metricnames.NameLabel})

	localAggregates  = &configGauges{vecs: []*prometheus.GaugeVec{localAggregateValue, localAggregateTimestamp}, timestamp: localAggregateTimestamp}
	globalAggregates = &configGauges{vecs: []*prometheus.GaugeVec{globalAggregateValue, globalAggregateRound, globalAggregateTimestamp}, timestamp: globalAggregateTimestamp}
)

// configGauges tracks the loaded config names so that series of configs
// removed on refresh can be dropped.
type configGauges struct {
	mu        sync.Mutex
	names     map[string]struct{}
	vecs      []*prometheus.GaugeVec
	timestamp *prometheus.GaugeVec
}

func (g *configGauges) loaded(name string) bool {
	_, ok := g.names[name]
	return ok
}

// setConfigs drops series of names not in the loaded config set, and gives
// newly loaded names a timestamp of now (no value) so a config that never
// produces an aggregate still shows a growing delay.
func (g *configGauges) setConfigs(names []string, now time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()

	loaded := make(map[string]struct{}, len(names))
	for _, name := range names {
		loaded[name] = struct{}{}
	}

	for name := range g.names {
		if _, ok := loaded[name]; ok {
			continue
		}
		for _, vec := range g.vecs {
			vec.DeleteLabelValues(name)
		}
		delete(g.names, name)
	}

	for name := range loaded {
		if _, ok := g.names[name]; ok {
			continue
		}
		g.timestamp.WithLabelValues(name).Set(unixSeconds(now))
		if g.names == nil {
			g.names = make(map[string]struct{})
		}
		g.names[name] = struct{}{}
	}
}

func unixSeconds(t time.Time) float64 {
	return float64(t.UnixNano()) / 1e9
}

// SetLocalAggregateConfigs syncs local aggregate series with loaded config names.
func SetLocalAggregateConfigs(names []string) {
	localAggregates.setConfigs(names, time.Now())
}

// SetGlobalAggregateConfigs syncs global aggregate series with loaded config names.
func SetGlobalAggregateConfigs(names []string) {
	globalAggregates.setConfigs(names, time.Now())
}

// SetLocalAggregate records a local aggregate; ignored for names not loaded
// (a late write from a stopped aggregator must not revive a removed config).
func SetLocalAggregate(name string, value int64, timestamp time.Time) {
	localAggregates.mu.Lock()
	defer localAggregates.mu.Unlock()
	if !localAggregates.loaded(name) {
		return
	}
	localAggregateValue.WithLabelValues(name).Set(float64(value))
	localAggregateTimestamp.WithLabelValues(name).Set(unixSeconds(timestamp))
}

// SetGlobalAggregate records a global aggregate; ignored for names not loaded.
func SetGlobalAggregate(name string, value int64, round int32, timestamp time.Time) {
	globalAggregates.mu.Lock()
	defer globalAggregates.mu.Unlock()
	if !globalAggregates.loaded(name) {
		return
	}
	globalAggregateValue.WithLabelValues(name).Set(float64(value))
	globalAggregateRound.WithLabelValues(name).Set(float64(round))
	globalAggregateTimestamp.WithLabelValues(name).Set(unixSeconds(timestamp))
}
