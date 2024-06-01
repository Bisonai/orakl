package tests

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/common/keys"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/binance"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/bithumb"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/btse"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/bybit"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/coinbase"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/coinone"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/crypto"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/korbit"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/kucoin"
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
			FeedID:    1,
			Value:     10000,
			Timestamp: &testTimestamps[0],
		},
		{
			FeedID:    1,
			Value:     10001,
			Timestamp: &testTimestamps[1],
		},
		{
			FeedID:    2,
			Value:     20000,
			Timestamp: &testTimestamps[2],
		},
		{
			FeedID:    2,
			Value:     20001,
			Timestamp: &testTimestamps[3],
		},
	}

	err := common.StoreFeeds(ctx, feedData)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	latestFeed1, err := db.GetObject[common.FeedData](ctx, keys.LatestFeedDataKey(1))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if latestFeed1.Value != 10001 {
		t.Errorf("expected value 10001, got %f", latestFeed1.Value)
	}

	latestFeed2, err := db.GetObject[common.FeedData](ctx, keys.LatestFeedDataKey(2))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if latestFeed2.Value != 20001 {
		t.Errorf("expected value 20001, got %f", latestFeed2.Value)
	}

	buffer, err := db.PopAllObject[common.FeedData](ctx, keys.FeedDataBufferKey())
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

	t.Run("TestMessageToStructKucoin", func(t *testing.T) {
		jsonStr := `{
			"type": "message",
			"topic": "/market/ticker:BTC-USDT",
			"subject": "trade.ticker",
			"data": {
			  "sequence": "1545896668986",
			  "price": "0.08",
			  "size": "0.011",
			  "bestAsk": "0.08",
			  "bestAskSize": "0.18",
			  "bestBid": "0.049",
			  "bestBidSize": "0.036",
			  "Time": 1704873323416
			}
		  }`

		var result map[string]any
		err := json.Unmarshal([]byte(jsonStr), &result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		data, err := common.MessageToStruct[kucoin.Raw](result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		assert.Equal(t, "/market/ticker:BTC-USDT", data.Topic)
		assert.Equal(t, "0.08", data.Data.Price)
		assert.Equal(t, int64(1704873323416), data.Data.Time)
	})

	t.Run("TestMessageToStructBybit", func(t *testing.T) {
		jsonStr := `{
			"topic": "mockTopic",
			"type": "mockType",
			"data": {
			  "symbol": "mockSymbol",
			  "tickDirection": "mockTickDirection",
			  "price24hPcnt": "1.23",
			  "lastPrice": "456.78",
			  "prevPrice24h": "123.45",
			  "highPrice24h": "789.01",
			  "lowPrice24h": "234.56",
			  "prevPrice1h": "345.67",
			  "markPrice": "890.12",
			  "indexPrice": "456.78",
			  "openInterest": "567.89",
			  "openInterestValue": "901.23",
			  "turnover24h": "234.56",
			  "volume24h": "345.67",
			  "nextFundingTime": "2022-12-31T23:59:59Z",
			  "fundingRate": "0.01",
			  "bid1Price": "123.45",
			  "bid1Size": "234.56",
			  "ask1Price": "345.67",
			  "ask1Size": "456.78"
			},
			"cs": 123456789012345,
			"ts": 234567890123456
		  }`

		var result map[string]any
		err := json.Unmarshal([]byte(jsonStr), &result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		data, err := common.MessageToStruct[bybit.Response](result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		assert.Equal(t, "mockSymbol", *data.Data.Symbol)
		assert.Equal(t, "456.78", *data.Data.LastPrice)
		assert.Equal(t, int64(234567890123456), data.Ts)
	})

	t.Run("TestMessageToStructCryptoDotCom", func(t *testing.T) {
		jsonStr := `{
			"id": -1,
			"method": "subscribe",
			"code": 0,
			"result": {
			  "channel": "ticker",
			  "instrument_name": "ADA_USDT",
			  "subscription": "ticker.ADA_USDT",
			  "data": [
				{
				  "h": "0.45575",
				  "l": "0.44387",
				  "a": "0.44878",
				  "i": "ADA_USDT",
				  "v": "2900036",
				  "vv": "1303481.10",
				  "oi": "0",
				  "c": "0.0016",
				  "b": "0.44870",
				  "k": "0.44880",
				  "t": 1717223914135
				}
			  ]
			}
		  }`
		var result map[string]any
		err := json.Unmarshal([]byte(jsonStr), &result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		data, err := common.MessageToStruct[crypto.Response](result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		assert.Equal(t, "ticker", data.Result.Channel)
		assert.Equal(t, "ADA_USDT", data.Result.InstrumentName)
		assert.Equal(t, int64(1717223914135), data.Result.Data[0].Timestamp)
		assert.Equal(t, "0.44878", *data.Result.Data[0].LastTradePrice)
	})

	t.Run("TestMessageToStructBtse", func(t *testing.T) {
		jsonStr := `{
			"topic": "tradeHistoryApi:ADA-USDT",
			"data": [
			  {
				"symbol": "ADA-USDT",
				"side": "SELL",
				"size": 122.4,
				"price": 0.44804,
				"tradeId": 62497538,
				"timestamp": 1717227427438
			  }
			]
		  }`

		var result map[string]any
		err := json.Unmarshal([]byte(jsonStr), &result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		data, err := common.MessageToStruct[btse.Response](result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		assert.Equal(t, "ADA-USDT", data.Data[0].Symbol)
		assert.Equal(t, float64(0.44804), data.Data[0].Price)
		assert.Equal(t, int64(1717227427438), data.Data[0].Timestamp)
	})

	t.Run("TestMessageToStructBithumb", func(t *testing.T) {
		txJsonStr := `{
			"type": "transaction",
			"content": {
				"list": [
					{
						"symbol": "BTC_KRW",
						"buySellGb": "1",
						"contPrice": "10579000",
						"contQty": "0.01",
						"contAmt": "105790.00",
						"contDtm": "2020-01-29 12:24:18.830039",
						"updn": "dn"
					},
					{
						"symbol": "ETH_KRW",
						"buySellGb": "2",
						"contPrice": "200000",
						"contQty": "0.05",
						"contAmt": "10000.00",
						"contDtm": "2020-01-29 12:24:18.830039",
						"updn": "up"
					}
				]
			}
		}`

		tickerJsonStr := `{
			"type": "ticker",
			"content": {
				"symbol": "BTC_KRW",
				"tickType": "1H",
				"date": "20240601",
				"time": "171451",
				"openPrice": "1227",
				"closePrice": "1224",
				"lowPrice": "1223",
				"highPrice": "1230",
				"value": "22271989.6261801699999998",
				"volume": "18172.56112368162601626",
				"sellVolume": "10920.87235377",
				"buyVolume": "7251.68876991162601626",
				"prevClosePrice": "1211",
				"chgRate": "-0.24",
				"chgAmt": "-3",
				"volumePower": "66.4"
			}
		}`

		var txResult map[string]any
		err := json.Unmarshal([]byte(txJsonStr), &txResult)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		txData, err := common.MessageToStruct[bithumb.TransactionResponse](txResult)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		assert.Equal(t, "BTC_KRW", txData.Content.List[0].Symbol)
		assert.Equal(t, "10579000", txData.Content.List[0].ContPrice)
		assert.Equal(t, "2020-01-29 12:24:18.830039", txData.Content.List[0].ContDtm)

		var tickerResult map[string]any
		err = json.Unmarshal([]byte(tickerJsonStr), &tickerResult)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		tickerData, err := common.MessageToStruct[bithumb.TickerResponse](tickerResult)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		assert.Equal(t, "BTC_KRW", tickerData.Content.Symbol)
		assert.Equal(t, "20240601", tickerData.Content.Date)
		assert.Equal(t, "171451", tickerData.Content.Time)

	})

}
