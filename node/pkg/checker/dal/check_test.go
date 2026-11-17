package dal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func dalServer(t *testing.T, data *[]OutgoingSubmissionData) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(*data)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestCheckDalExportsAndDropsSymbols(t *testing.T) {
	full := func(symbol, ts string) OutgoingSubmissionData {
		return OutgoingSubmissionData{Symbol: symbol, Value: "1", AggregateTime: ts, Proof: "p", FeedHash: "h", Decimals: "8"}
	}
	data := []OutgoingSubmissionData{
		full("BTC-USDT", "1700000000000"),
		full("ETH-USDT", "1700000001500"),
		full("JOY-USDT", "1700000000000"), // delisted, not exported
		{Symbol: "KAIA-USDT", AggregateTime: "1700000000000"},
	}
	srv := dalServer(t, &data)
	restAggregateTimestampGauge.Reset()
	symbols := map[string]struct{}{}

	if err := checkDal(srv.URL, "key", symbols); err != nil {
		t.Fatal(err)
	}
	if v := testutil.ToFloat64(restAggregateTimestampGauge.WithLabelValues("ETH-USDT")); v != 1700000001.5 {
		t.Fatalf("ETH timestamp: got %v", v)
	}
	if v := testutil.ToFloat64(restEmptyFeedsGauge); v != 1 {
		t.Fatalf("empty feeds: got %v, want 1", v)
	}
	if n := testutil.CollectAndCount(restAggregateTimestampGauge); n != 3 {
		t.Fatalf("expected 3 series (delisted skipped), got %d", n)
	}

	data = data[:1]
	if err := checkDal(srv.URL, "key", symbols); err != nil {
		t.Fatal(err)
	}
	if n := testutil.CollectAndCount(restAggregateTimestampGauge); n != 1 {
		t.Fatalf("expected dropped symbols removed, got %d series", n)
	}
}

func TestCheckDalErrorKeepsLastSuccess(t *testing.T) {
	restLastSuccessGauge.Set(42)
	if err := checkDal("http://127.0.0.1:0", "key", map[string]struct{}{}); err == nil {
		t.Fatal("expected error")
	}
	if v := testutil.ToFloat64(restLastSuccessGauge); v != 42 {
		t.Fatalf("last success changed on error: %v", v)
	}
}
