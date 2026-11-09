package metrics

import (
	"testing"
	"time"

	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordRaftElectionStarted(t *testing.T) {
	timeout := raftElectionsStarted.WithLabelValues(RaftElectionReasonTimeout)
	before := testutil.ToFloat64(timeout)

	RecordRaftElectionStarted(RaftElectionReasonTimeout)

	assert.Equal(t, before+1, testutil.ToFloat64(timeout))
}

func TestRecordRaftLeaderSilence(t *testing.T) {
	// bucket upper bound -> cumulative count, from the registry /metrics serves
	read := func() (uint64, map[float64]uint64) {
		families, err := prometheus.DefaultGatherer.Gather()
		require.NoError(t, err)
		for _, family := range families {
			if family.GetName() == metricnames.RaftLeaderSilence {
				histogram := family.GetMetric()[0].GetHistogram()
				buckets := map[float64]uint64{}
				for _, bucket := range histogram.GetBucket() {
					buckets[bucket.GetUpperBound()] = bucket.GetCumulativeCount()
				}
				return histogram.GetSampleCount(), buckets
			}
		}
		t.Fatalf("%s not registered", metricnames.RaftLeaderSilence)
		return 0, nil
	}

	count, buckets := read()

	RecordRaftLeaderSilence(1800 * time.Millisecond)

	newCount, newBuckets := read()
	assert.Equal(t, count+1, newCount)
	assert.Equal(t, buckets[1.5], newBuckets[1.5], "1.8 s is above the 1.5 s bucket")
	assert.Equal(t, buckets[2]+1, newBuckets[2], "1.8 s is within the 2 s bucket")
	for _, bound := range []float64{0.05, 2.5, 3, 10} {
		assert.Contains(t, newBuckets, bound)
	}
}
