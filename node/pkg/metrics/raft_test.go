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
	for _, reason := range []string{RaftElectionReasonTimeout, RaftElectionReasonDisconnect, RaftElectionReasonShutdown} {
		counter := raftElectionsStarted.WithLabelValues(reason)
		before := testutil.ToFloat64(counter)

		RecordRaftElectionStarted(reason)

		assert.Equal(t, before+1, testutil.ToFloat64(counter), reason)
	}
}

func TestRecordRaftLeaderChangeSilence(t *testing.T) {
	read := func() (uint64, float64) {
		families, err := prometheus.DefaultGatherer.Gather()
		require.NoError(t, err)
		for _, family := range families {
			if family.GetName() == metricnames.RaftLeaderChangeSilence {
				histogram := family.GetMetric()[0].GetHistogram()
				return histogram.GetSampleCount(), histogram.GetSampleSum()
			}
		}
		t.Fatalf("%s not registered", metricnames.RaftLeaderChangeSilence)
		return 0, 0
	}
	count, sum := read()

	RecordRaftLeaderChangeSilence(20 * time.Second)

	newCount, newSum := read()
	assert.Equal(t, count+1, newCount)
	assert.InDelta(t, sum+20, newSum, 1e-9)
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
