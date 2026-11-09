package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRecordP2PPeerByType(t *testing.T) {
	oracleConnects := testutil.ToFloat64(p2pPeerConnects.WithLabelValues(P2PPeerTypeOracle))
	bootConnects := testutil.ToFloat64(p2pPeerConnects.WithLabelValues(P2PPeerTypeBoot))
	oracleDisconnects := testutil.ToFloat64(p2pPeerDisconnects.WithLabelValues(P2PPeerTypeOracle))
	bootDisconnects := testutil.ToFloat64(p2pPeerDisconnects.WithLabelValues(P2PPeerTypeBoot))

	RecordP2PPeerConnect(P2PPeerTypeOracle)
	RecordP2PPeerDisconnect(P2PPeerTypeBoot)

	assert.Equal(t, oracleConnects+1, testutil.ToFloat64(p2pPeerConnects.WithLabelValues(P2PPeerTypeOracle)))
	assert.Equal(t, bootConnects, testutil.ToFloat64(p2pPeerConnects.WithLabelValues(P2PPeerTypeBoot)))
	assert.Equal(t, oracleDisconnects, testutil.ToFloat64(p2pPeerDisconnects.WithLabelValues(P2PPeerTypeOracle)))
	assert.Equal(t, bootDisconnects+1, testutil.ToFloat64(p2pPeerDisconnects.WithLabelValues(P2PPeerTypeBoot)))
}

func TestRecordP2PPublished(t *testing.T) {
	t.Cleanup(func() {
		p2pPublishedMessages.Reset()
		p2pPublishedBytes.Reset()
	})

	RecordP2PPublished("TEST-A", 10)
	RecordP2PPublished("TEST-A", 5)
	RecordP2PPublished("TEST-B", 7)

	assert.Equal(t, float64(2), testutil.ToFloat64(p2pPublishedMessages.WithLabelValues("TEST-A")))
	assert.Equal(t, float64(15), testutil.ToFloat64(p2pPublishedBytes.WithLabelValues("TEST-A")))
	assert.Equal(t, float64(1), testutil.ToFloat64(p2pPublishedMessages.WithLabelValues("TEST-B")))
	assert.Equal(t, float64(7), testutil.ToFloat64(p2pPublishedBytes.WithLabelValues("TEST-B")))
}
