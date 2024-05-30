package tests

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/binance"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/coinbase"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/coinone"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/korbit"
	"github.com/stretchr/testify/assert"
)

var testFeeds = []common.Feed{
	{
		ID:         1,
		Name:       "binance-wss-BTC-USDT",
		Definition: json.RawMessage(`{"type": "wss", "provider": "binance", "base": "btc", "quote": "usdt"}`),
		ConfigID:   1,
	},
	{
		ID:         2,
		Name:       "coinbase-wss-ADA-USDT",
		Definition: json.RawMessage(`{"type": "wss", "provider": "coinbase", "base": "ada", "quote": "usdt"}`),
		ConfigID:   2,
	},
	{
		ID:         3,
		Name:       "coinone-wss-BTC-KRW",
		Definition: json.RawMessage(`{"type": "wss", "provider": "coinone", "base": "btc", "quote": "krw"}`),
		ConfigID:   3,
	},
	{
		ID:         4,
		Name:       "korbit-wss-BORA-KRW",
		Definition: json.RawMessage(`{"type": "wss", "provider": "korbit", "base": "bora", "quote": "krw"}`),
		ConfigID:   4,
	},
}

func TestGetWssFeedMap(t *testing.T) {
	feedMaps := common.GetWssFeedMap(testFeeds)
	if len(feedMaps) != 4 {
		t.Errorf("expected 4 feed maps, got %d", len(feedMaps))
	}

	for _, feed := range testFeeds {
		raw := strings.Split(feed.Name, "-")
		if len(raw) != 4 {
			t.Errorf("expected 4 parts, got %d", len(raw))
		}

		provider := strings.ToLower(raw[0])
		base := strings.ToUpper(raw[2])
		quote := strings.ToUpper(raw[3])
		combinedName := base + quote
		separatedName := base + "-" + quote

		if _, exists := feedMaps[provider]; !exists {
			t.Errorf("provider %s not found", provider)
		}
		if _, exists := feedMaps[provider].Combined[combinedName]; !exists {
			t.Errorf("combined feed %s not found", combinedName)
		}
		if _, exists := feedMaps[provider].Separated[separatedName]; !exists {
			t.Errorf("separated feed %s not found", separatedName)
		}
	}
}

func TestStoreFeeds(t *testing.T) {
	ctx := context.Background()
	testTimestamps := []time.Time{
		time.Now(),
		time.Now().Add(time.Second),
		time.Now().Add(time.Second * 2),
		time.Now().Add(time.Second * 3),
	}

	feedData := []common.FeedData{
		{
			FeedId:    1,
			Value:     10000,
			Timestamp: &testTimestamps[0],
		},
		{
			FeedId:    1,
			Value:     10001,
			Timestamp: &testTimestamps[1],
		},
		{
			FeedId:    2,
			Value:     20000,
			Timestamp: &testTimestamps[2],
		},
		{
			FeedId:    2,
			Value:     20001,
			Timestamp: &testTimestamps[3],
		},
	}

	err := common.StoreFeeds(ctx, feedData)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	latestFeed1, err := db.GetObject[common.FeedData](ctx, "latestFeedData:1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if latestFeed1.Value != 10001 {
		t.Errorf("expected value 10001, got %f", latestFeed1.Value)
	}

	latestFeed2, err := db.GetObject[common.FeedData](ctx, "latestFeedData:2")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if latestFeed2.Value != 20001 {
		t.Errorf("expected value 20001, got %f", latestFeed2.Value)
	}

	buffer, err := db.PopAllObject[common.FeedData](ctx, "feedDataBuffer")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(buffer) != 4 {
		t.Errorf("expected 4 buffer data, got %d", len(buffer))
	}
}

func TestPriceStringToFloat64(t *testing.T) {
	price := "10000.123400"
	expected := 1000012340000.0
	result, err := common.PriceStringToFloat64(price)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %f, got %f", expected, result)
	}
}

func TestMessageToStruct(t *testing.T) {
	t.Run("TestMessageToStructBinance", func(t *testing.T) {
		jsonStr := `{
			"e": "24hrMiniTicker",
			"E": 1672515782136,
			"s": "BNBBTC",
			"c": "0.0025",
			"o": "0.0010",
			"h": "0.0025",
			"l": "0.0010",
			"v": "10000",
			"q": "18"
		  }`

		var result map[string]any
		err := json.Unmarshal([]byte(jsonStr), &result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		data, err := common.MessageToStruct[binance.MiniTicker](result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		assert.Equal(t, "BNBBTC", data.Symbol)
		assert.Equal(t, "0.0025", data.Price)
		assert.Equal(t, int64(1672515782136), data.EventTime)
	})

	t.Run("TestMessageToStructCoinbase", func(t *testing.T) {
		jsonStr := `{
			"type": "ticker",
			"sequence": 123456789,
			"product_id": "BTC-USD",
			"price": "50000.00",
			"open_24h": "48000.00",
			"volume_24h": "10000",
			"low_24h": "47000.00",
			"high_24h": "51000.00",
			"volume_30d": "300000",
			"best_bid": "49999.00",
			"best_bid_size": "0.5",
			"best_ask": "50001.00",
			"best_ask_size": "0.5",
			"side": "buy",
			"time": "2022-01-01T00:00:00Z",
			"trade_id": 1234,
			"last_size": "0.01"
		}`

		var result map[string]any
		err := json.Unmarshal([]byte(jsonStr), &result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		data, err := common.MessageToStruct[coinbase.Ticker](result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		assert.Equal(t, "BTC-USD", data.ProductID)
		assert.Equal(t, "50000.00", data.Price)
		assert.Equal(t, "2022-01-01T00:00:00Z", data.Time)
	})

	t.Run("TestMessageToStructCoinone", func(t *testing.T) {
		jsonStr := `
			{
				"r":"DATA",
				"c":"TICKER",
				"d":{
				  "qc":"KRW",
				  "tc":"XRP",
				  "t":1693560378928,
				  "qv":"55827441390.8456",
				  "tv":"79912892.7741579",
				  "fi":"698.7",
				  "lo":"683.9",
				  "hi":"699.5",
				  "la":"687.9",
				  "vp":"100",
				  "abp":"688.3",
				  "abq":"84992.9448",
				  "bbp":"687.8",
				  "bbq":"13861.6179",
				  "i":"1693560378928001",
				  "yfi":"716.9",
				  "ylo":"690.4",
				  "yhi":"717.5",
				  "yla":"698.7",
				  "yqv":"41616318229.6505",
				  "ytv":"58248252.35151376"
				}
			  }
		`

		var result map[string]any
		err := json.Unmarshal([]byte(jsonStr), &result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		data, err := common.MessageToStruct[coinone.Raw](result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		assert.Equal(t, "KRW", data.Data.QuoteCurrency)
		assert.Equal(t, "XRP", data.Data.TargetCurrency)
		assert.Equal(t, int64(1693560378928), data.Data.Timestamp)
		assert.Equal(t, "687.9", data.Data.Last)
	})

	t.Run("TestMessageToStructKorbit", func(t *testing.T) {
		jsonStr := `{
			"accessToken": null,
			"event": "korbit:push-ticker",
			"timestamp" : 1389678052000,
			"data":
			  {
				"channel": "ticker",
				"currency_pair": "btc_krw",
				"timestamp": 1558590089274,
				"last": "9198500.1235789",
				"open": "9500000.3445783",
				"bid": "9192500.4578344",
				"ask": "9198000.32148556",
				"low": "9171500.23785685",
				"high": "9599000.34876458",
				"volume": "1539.18571988",
				"change": "-301500.234578934"
			}
		  }`

		var result map[string]any
		err := json.Unmarshal([]byte(jsonStr), &result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		data, err := common.MessageToStruct[korbit.Raw](result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		assert.Equal(t, "btc_krw", data.Data.CurrencyPair)
		assert.Equal(t, int64(1558590089274), data.Data.Timestamp)
		assert.Equal(t, "9198500.1235789", data.Data.Last)
	})
}
