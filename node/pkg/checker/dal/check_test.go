package dal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/checker"
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

	if err := checkDal(srv.URL, "key", nil, symbols); err != nil {
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
	if err := checkDal(srv.URL, "key", nil, symbols); err != nil {
		t.Fatal(err)
	}
	if n := testutil.CollectAndCount(restAggregateTimestampGauge); n != 1 {
		t.Fatalf("expected dropped symbols removed, got %d series", n)
	}
}

func TestCheckDalErrorKeepsLastSuccess(t *testing.T) {
	restLastSuccessGauge.Set(42)
	if err := checkDal("http://127.0.0.1:0", "key", nil, map[string]struct{}{}); err == nil {
		t.Fatal("expected error")
	}
	if v := testutil.ToFloat64(restLastSuccessGauge); v != 42 {
		t.Fatalf("last success changed on error: %v", v)
	}
}

func fullEntry(symbol, ts string) OutgoingSubmissionData {
	return OutgoingSubmissionData{Symbol: symbol, Value: "1", AggregateTime: ts, Proof: "p", FeedHash: "h", Decimals: "8"}
}

func expectedOf(names ...string) map[string]struct{} {
	e := &expectedSymbols{}
	configs := make([]Config, len(names))
	for i, n := range names {
		configs[i] = Config{Name: n}
	}
	e.set(configs)
	return e.get()
}

func TestCheckDalEmptyResponseIsFailure(t *testing.T) {
	data := []OutgoingSubmissionData{fullEntry("BTC-USDT", "1700000000000"), fullEntry("ETH-USDT", "1700000000000")}
	srv := dalServer(t, &data)
	restAggregateTimestampGauge.Reset()
	expected := expectedOf("BTC-USDT", "ETH-USDT", checker.SymbolsToBeDelisted[0]) // delisted symbols are not expected
	symbols := map[string]struct{}{}

	if err := checkDal(srv.URL, "key", expected, symbols); err != nil {
		t.Fatal(err)
	}
	if v := testutil.ToFloat64(restMissingFeedsGauge); v != 0 {
		t.Fatalf("missing feeds: got %v, want 0 (delisted not expected)", v)
	}

	symbols["GONE-USDT"] = struct{}{} // exported earlier, no longer configured
	restAggregateTimestampGauge.WithLabelValues("GONE-USDT").Set(1)
	restLastSuccessGauge.Set(42)
	data = []OutgoingSubmissionData{}
	if err := checkDal(srv.URL, "key", expected, symbols); err == nil {
		t.Fatal("expected error for empty response")
	}
	if v := testutil.ToFloat64(restLastSuccessGauge); v != 42 {
		t.Fatalf("last success changed on empty response: %v", v)
	}
	if v := testutil.ToFloat64(restMissingFeedsGauge); v != 2 {
		t.Fatalf("missing feeds: got %v, want 2", v)
	}
	if n := testutil.CollectAndCount(restAggregateTimestampGauge); n != 3 {
		t.Fatalf("failed check must keep all series, got %d", n)
	}
	if v := testutil.ToFloat64(restAggregateTimestampGauge.WithLabelValues("BTC-USDT")); v != 1700000000 {
		t.Fatalf("BTC timestamp: got %v", v)
	}
}

func TestExportLastUpdatesDropsUnexpectedSymbols(t *testing.T) {
	wsLastPushGauge.Reset()
	u := &UpdateTimes{lastUpdates: map[string]time.Time{}}
	u.Store("BTC-USDT", time.UnixMilli(1700000000000))
	u.Store("BTT-KRW", time.UnixMilli(1700000001500))
	u.Store(checker.SymbolsToBeDelisted[0], time.UnixMilli(1700000000000))

	u.exportLastUpdates(expectedOf("BTC-USDT", "BTT-KRW", checker.SymbolsToBeDelisted[0]))
	if n := testutil.CollectAndCount(wsLastPushGauge); n != 2 {
		t.Fatalf("expected BTC + BTT series (delisted skipped), got %d", n)
	}
	if v := testutil.ToFloat64(wsLastPushGauge.WithLabelValues("BTT-KRW")); v != 1700000001.5 {
		t.Fatalf("BTT push time: got %v", v)
	}

	// BTT-KRW removed from config on reload: its series and entry are dropped.
	u.exportLastUpdates(expectedOf("BTC-USDT"))
	if n := testutil.CollectAndCount(wsLastPushGauge); n != 1 {
		t.Fatalf("expected only BTC series after reload, got %d", n)
	}
	if _, ok := u.lastUpdates["BTT-KRW"]; ok {
		t.Fatal("removed symbol should be dropped from lastUpdates")
	}
	if v := testutil.ToFloat64(wsLastPushGauge.WithLabelValues("BTC-USDT")); v != 1700000000 {
		t.Fatalf("BTC push time: got %v", v)
	}
}

func TestCheckDalMissingConfiguredSymbol(t *testing.T) {
	data := []OutgoingSubmissionData{
		fullEntry("BTC-USDT", "1700000000000"),
		fullEntry("ETH-USDT", "1700000000000"),
		fullEntry("OLD-USDT", "1700000000000"),
	}
	srv := dalServer(t, &data)
	restAggregateTimestampGauge.Reset()
	symbols := map[string]struct{}{}

	if err := checkDal(srv.URL, "key", expectedOf("BTC-USDT", "ETH-USDT", "OLD-USDT", "NEW-USDT"), symbols); err != nil {
		t.Fatal(err)
	}
	if v := testutil.ToFloat64(restMissingFeedsGauge); v != 1 {
		t.Fatalf("missing feeds: got %v, want 1 (NEW-USDT)", v)
	}

	// ETH-USDT stops being served while still configured; OLD-USDT is removed
	// from config and no longer served.
	restLastSuccessGauge.Set(42)
	data = data[:1]
	if err := checkDal(srv.URL, "key", expectedOf("BTC-USDT", "ETH-USDT", "NEW-USDT"), symbols); err != nil {
		t.Fatal(err)
	}
	if v := testutil.ToFloat64(restMissingFeedsGauge); v != 2 {
		t.Fatalf("missing feeds: got %v, want 2", v)
	}
	if v := testutil.ToFloat64(restLastSuccessGauge); v == 42 {
		t.Fatal("partial response should still update last success")
	}
	if n := testutil.CollectAndCount(restAggregateTimestampGauge); n != 2 {
		t.Fatalf("expected BTC + stale ETH series, got %d", n)
	}
	if _, ok := symbols["OLD-USDT"]; ok {
		t.Fatal("removed-from-config symbol should be dropped")
	}
	if _, ok := symbols["ETH-USDT"]; !ok {
		t.Fatal("configured symbol should be kept")
	}
}
