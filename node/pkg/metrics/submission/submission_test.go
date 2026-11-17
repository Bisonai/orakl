package submission

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordSubmission(t *testing.T) {
	counter := submissions.WithLabelValues(PathDirectFallback, PurposeSubmit)
	before := testutil.ToFloat64(counter)

	RecordSubmission(PathDirectFallback, PurposeSubmit)

	assert.Equal(t, before+1, testutil.ToFloat64(counter))
}

func TestRecordDelegatorError(t *testing.T) {
	counter := delegatorErrors.WithLabelValues(ReasonHTTPStatus, PurposeRotate)
	before := testutil.ToFloat64(counter)

	RecordDelegatorError(ReasonHTTPStatus, PurposeRotate)

	assert.Equal(t, before+1, testutil.ToFloat64(counter))
}

// After first use, every bounded series is exported (at 0 until it happens).
func TestRegisteredWithZeroSeries(t *testing.T) {
	RecordSubmission(PathDelegated, PurposeSubmit)

	n, err := testutil.GatherAndCount(prometheus.DefaultGatherer, "orakl_node_reporter_submissions_total", "orakl_node_reporter_delegator_errors_total")
	require.NoError(t, err)
	assert.Equal(t, 2*3+2*3, n)
}
