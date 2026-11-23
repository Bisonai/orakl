package signer

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordRotationFailure(t *testing.T) {
	counter := rotationFailures.WithLabelValues(ReasonNotSent)
	before := testutil.ToFloat64(counter)

	RecordRotationFailure(ReasonNotSent)

	assert.Equal(t, before+1, testutil.ToFloat64(counter))
}

// After Register, every bounded series is exported (at 0 until it happens).
func TestRegisteredWithZeroSeries(t *testing.T) {
	Register()

	n, err := testutil.GatherAndCount(prometheus.DefaultGatherer, "orakl_node_signer_rotation_failures_total")
	require.NoError(t, err)
	assert.Equal(t, 2, n)
}
