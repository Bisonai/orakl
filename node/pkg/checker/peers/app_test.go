package peers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestCheckExportsMaxPeerCount(t *testing.T) {
	counts := []string{`{"Count":3}`, `{"Count":5}`, `{"Count":4}`, `{"Count":5}`, `{"Count":2}`}
	i := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(counts[i%len(counts)]))
		i++
	}))
	defer srv.Close()
	t.Setenv("ORAKL_NODE_ADMIN_URL", srv.URL)

	check()
	if v := testutil.ToFloat64(peerCountGauge); v != 5 {
		t.Fatalf("peer count: got %v, want 5", v)
	}
}

func TestCheckAllSamplesFailCountsError(t *testing.T) {
	t.Setenv("ORAKL_NODE_ADMIN_URL", "http://127.0.0.1:0")
	peerCountGauge.Set(4)
	before := testutil.ToFloat64(errorsCounter)

	check()
	if d := testutil.ToFloat64(errorsCounter) - before; d != 1 {
		t.Fatalf("expected 1 error, got %v", d)
	}
	if v := testutil.ToFloat64(peerCountGauge); v != 4 {
		t.Fatalf("peer count must not drop to 0 on fetch failure, got %v", v)
	}
}
