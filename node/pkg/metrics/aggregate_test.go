package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestSetLocalAggregate(t *testing.T) {
	ts := time.Unix(1700000000, 500_000_000)
	SetLocalAggregate("LOCAL-SET", 12345, ts)
	t.Cleanup(func() { SetLocalAggregateConfigs(nil) })

	assert.Equal(t, float64(12345), testutil.ToFloat64(localAggregateValue.WithLabelValues("LOCAL-SET")))
	assert.Equal(t, 1700000000.5, testutil.ToFloat64(localAggregateTimestamp.WithLabelValues("LOCAL-SET")))
}

func TestSetGlobalAggregate(t *testing.T) {
	ts := time.Unix(1700000001, 0)
	SetGlobalAggregate("GLOBAL-SET", 67890, 42, ts)
	t.Cleanup(func() { SetGlobalAggregateConfigs(nil) })

	assert.Equal(t, float64(67890), testutil.ToFloat64(globalAggregateValue.WithLabelValues("GLOBAL-SET")))
	assert.Equal(t, float64(42), testutil.ToFloat64(globalAggregateRound.WithLabelValues("GLOBAL-SET")))
	assert.Equal(t, float64(1700000001), testutil.ToFloat64(globalAggregateTimestamp.WithLabelValues("GLOBAL-SET")))
}

func TestSetConfigsDropsRemovedAndSeedsNew(t *testing.T) {
	t.Cleanup(func() { SetGlobalAggregateConfigs(nil) })

	SetGlobalAggregate("KEEP", 1, 1, time.Unix(100, 0))
	SetGlobalAggregate("REMOVED", 2, 2, time.Unix(200, 0))

	now := time.Unix(300, 0)
	globalAggregates.setConfigs([]string{"KEEP", "NEW"}, now)

	// kept config keeps its latest values
	assert.Equal(t, float64(1), testutil.ToFloat64(globalAggregateValue.WithLabelValues("KEEP")))
	assert.Equal(t, float64(100), testutil.ToFloat64(globalAggregateTimestamp.WithLabelValues("KEEP")))

	// removed config has no series left
	assert.Equal(t, 1, testutil.CollectAndCount(globalAggregateValue))
	assert.Equal(t, 1, testutil.CollectAndCount(globalAggregateRound))
	assert.False(t, globalAggregateValue.DeleteLabelValues("REMOVED"))

	// new config gets only a timestamp of load time
	assert.Equal(t, float64(300), testutil.ToFloat64(globalAggregateTimestamp.WithLabelValues("NEW")))
	assert.False(t, globalAggregateValue.DeleteLabelValues("NEW"))
}
