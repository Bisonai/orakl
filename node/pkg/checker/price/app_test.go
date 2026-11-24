package price

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestComparePricesSkipsZeroAndMissing(t *testing.T) {
	btc := baseAndQuote{base: "BTC", quote: "USDT"}
	eth := baseAndQuote{base: "ETH", quote: "USDT"}
	kaia := baseAndQuote{base: "KAIA", quote: "USDT"}
	got := comparePrices(
		map[baseAndQuote]float64{btc: 100, eth: 0, kaia: 1},
		map[baseAndQuote]float64{btc: 98, eth: 10},
	)
	if len(got) != 1 || got[btc] != [2]float64{100, 98} {
		t.Fatalf("unexpected compared prices: %v", got)
	}
}

func TestRecordPricesExportsDiffAndDropsOldPairs(t *testing.T) {
	btc := baseAndQuote{base: "BTC", quote: "USDT"}
	eth := baseAndQuote{base: "ETH", quote: "USDT"}
	recordPrices(map[baseAndQuote][2]float64{btc: {100, 98}, eth: {10, 10}}, time.Unix(1000, 0))
	recordPrices(map[baseAndQuote][2]float64{btc: {100, 98}}, time.Unix(2000, 0))

	if v := testutil.ToFloat64(diffRatioGauge.WithLabelValues("BTC-USDT")); v < 0.0199 || v > 0.0201 {
		t.Fatalf("BTC diff ratio: got %v, want 0.02", v)
	}
	if n := testutil.CollectAndCount(diffRatioGauge); n != 1 {
		t.Fatalf("expected 1 diff series, got %d", n)
	}
	if v := testutil.ToFloat64(lastSuccessGauge); v != 2000 {
		t.Fatalf("last success: got %v", v)
	}
}

func TestBinanceMid(t *testing.T) {
	got, err := binanceMid(binanceResponse{Symbol: "KAIAUSDT", BidPrice: "0.02750000", AskPrice: "0.02760000"})
	if err != nil || got < 0.027549 || got > 0.027551 {
		t.Fatalf("mid: got %v, %v; want 0.02755", got, err)
	}
	for _, r := range []binanceResponse{
		{BidPrice: "0.00000000", AskPrice: "1"},
		{BidPrice: "1", AskPrice: "0"},
		{BidPrice: "x", AskPrice: "1"},
	} {
		if _, err := binanceMid(r); err == nil {
			t.Fatalf("expected error for %+v", r)
		}
	}
}
