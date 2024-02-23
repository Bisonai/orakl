package fetcher

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/adapter"
	"bisonai.com/orakl/node/pkg/admin/tests"
	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/gofiber/fiber/v2"
)

var sampleData = []string{`{
	"name": "BNB-USDT",
	"feeds": [{
		"name": "Bybit-BNB-USDT",
		"definition": {
		  "url": "https://api.bybit.com/derivatives/v3/public/tickers?symbol=BNBUSDT",
		  "headers": {
			"Content-Type": "application/json"
		  },
		  "method": "GET",
		  "reducers": [
			{
			  "function": "PARSE",
			  "args": [
				"result",
				"list"
			  ]
			},
			{
			  "function": "INDEX",
			  "args": 0
			},
			{
			  "function": "PARSE",
			  "args": [
				"lastPrice"
			  ]
			},
			{
			  "function": "POW10",
			  "args": 8
			},
			{
			  "function": "ROUND"
			}
		  ]
		}
	  },
	  {
		"name": "Binance-BNB-USDT",
		"definition": {
		  "url": "https://api.binance.com/api/v3/avgPrice?symbol=BNBUSDT",
		  "headers": {
			"Content-Type": "application/json"
		  },
		  "method": "GET",
		  "reducers": [
			{
			  "function": "PARSE",
			  "args": [
				"price"
			  ]
			},
			{
			  "function": "POW10",
			  "args": 8
			},
			{
			  "function": "ROUND"
			}
		  ]
		}
	  },
	  {
		"name": "Coinbase-BNB-USDT",
		"definition": {
		  "url": "https://api.coinbase.com/v2/exchange-rates?currency=BNB",
		  "headers": {
			"Content-Type": "application/json"
		  },
		  "method": "GET",
		  "reducers": [
			{
			  "function": "PARSE",
			  "args": [
				"data",
				"rates",
				"USDT"
			  ]
			},
			{
			  "function": "POW10",
			  "args": 8
			},
			{
			  "function": "ROUND"
			}
		  ]
		}
	  },
	  {
		"name": "Kucoin-BNB-USDT",
		"definition": {
		  "url": "https://api.kucoin.com/api/v1/market/orderbook/level1?symbol=BNB-USDT",
		  "headers": {
			"Content-Type": "application/json"
		  },
		  "method": "GET",
		  "reducers": [
			{
			  "function": "PARSE",
			  "args": [
				"data",
				"price"
			  ]
			},
			{
			  "function": "POW10",
			  "args": 8
			},
			{
			  "function": "ROUND"
			}
		  ]
		}
	  },
	  {
		"name": "Btse-BNB-USDT",
		"definition": {
		  "url": "https://api.btse.com/spot/api/v3.2/price?symbol=BNB-USDT",
		  "headers": {
			"Content-Type": "application/json"
		  },
		  "method": "GET",
		  "reducers": [
			{
			  "function": "INDEX",
			  "args": 0
			},
			{
			  "function": "PARSE",
			  "args": [
				"indexPrice"
			  ]
			},
			{
			  "function": "POW10",
			  "args": 8
			},
			{
			  "function": "ROUND"
			}
		  ]
		}
	  },
	  {
		"name": "Gateio-BNB-USDT",
		"definition": {
		  "url": "https://api.gateio.ws/api/v4/spot/tickers?currency_pair=BNB_USDT",
		  "headers": {
			"Content-Type": "application/json"
		  },
		  "method": "GET",
		  "reducers": [
			{
			  "function": "INDEX",
			  "args": 0
			},
			{
			  "function": "PARSE",
			  "args": [
				"last"
			  ]
			},
			{
			  "function": "POW10",
			  "args": 8
			},
			{
			  "function": "ROUND"
			}
		  ]
		}
	  },
	  {
		"name": "Coinex-BNB-USDT",
		"definition": {
		  "url": "https://api.coinex.com/v1/market/ticker?market=BNBUSDT",
		  "headers": {
			"Content-Type": "application/json"
		  },
		  "method": "GET",
		  "reducers": [
			{
			  "function": "PARSE",
			  "args": [
				"data",
				"ticker",
				"last"
			  ]
			},
			{
			  "function": "POW10",
			  "args": 8
			},
			{
			  "function": "ROUND"
			}
		  ]
		}
	  }]
}`, `{
	"name": "BTC-USDT",
	"feeds": [
	  {
		"name": "Bybit-BTC-USDT",
		"definition": {
		  "url": "https://api.bybit.com/derivatives/v3/public/tickers?symbol=BTCUSDT",
		  "headers": { "Content-Type": "application/json" },
		  "method": "GET",
		  "reducers": [
			{ "function": "PARSE", "args": ["result", "list"] },
			{ "function": "INDEX", "args": 0 },
			{ "function": "PARSE", "args": ["lastPrice"] },
			{ "function": "POW10", "args": 8 },
			{ "function": "ROUND" }
		  ]
		}
	  },
	  {
		"name": "Binance-BTC-USDT",
		"definition": {
		  "url": "https://api.binance.com/api/v3/avgPrice?symbol=BTCUSDT",
		  "headers": { "Content-Type": "application/json" },
		  "method": "GET",
		  "reducers": [
			{ "function": "PARSE", "args": ["price"] },
			{ "function": "POW10", "args": 8 },
			{ "function": "ROUND" }
		  ]
		}
	  },
	  {
		"name": "Coinbase-BTC-USDT",
		"definition": {
		  "url": "https://api.coinbase.com/v2/exchange-rates?currency=BTC",
		  "headers": { "Content-Type": "application/json" },
		  "method": "GET",
		  "reducers": [
			{ "function": "PARSE", "args": ["data", "rates", "USDT"] },
			{ "function": "POW10", "args": 8 },
			{ "function": "ROUND" }
		  ]
		}
	  },
	  {
		"name": "Kucoin-BTC-USDT",
		"definition": {
		  "url": "https://api.kucoin.com/api/v1/market/orderbook/level1?symbol=BTC-USDT",
		  "headers": { "Content-Type": "application/json" },
		  "method": "GET",
		  "reducers": [
			{ "function": "PARSE", "args": ["data", "price"] },
			{ "function": "POW10", "args": 8 },
			{ "function": "ROUND" }
		  ]
		}
	  },
	  {
		"name": "Crypto-BTC-USDT",
		"definition": {
		  "url": "https://api.crypto.com/v2/public/get-ticker?instrument_name=BTC_USDT",
		  "headers": { "Content-Type": "application/json" },
		  "method": "GET",
		  "reducers": [
			{ "function": "PARSE", "args": ["result", "data"] },
			{ "function": "INDEX", "args": 0 },
			{ "function": "PARSE", "args": ["a"] },
			{ "function": "POW10", "args": 8 },
			{ "function": "ROUND" }
		  ]
		}
	  },
	  {
		"name": "Btse-BTC-USDT",
		"definition": {
		  "url": "https://api.btse.com/spot/api/v3.2/price?symbol=BTC-USDT",
		  "headers": { "Content-Type": "application/json" },
		  "method": "GET",
		  "reducers": [
			{ "function": "INDEX", "args": 0 },
			{ "function": "PARSE", "args": ["indexPrice"] },
			{ "function": "POW10", "args": 8 },
			{ "function": "ROUND" }
		  ]
		}
	  },
	  {
		"name": "Gateio-BTC-USDT",
		"definition": {
		  "url": "https://api.gateio.ws/api/v4/spot/tickers?currency_pair=BTC_USDT",
		  "headers": { "Content-Type": "application/json" },
		  "method": "GET",
		  "reducers": [
			{ "function": "INDEX", "args": 0 },
			{ "function": "PARSE", "args": ["last"] },
			{ "function": "POW10", "args": 8 },
			{ "function": "ROUND" }
		  ]
		}
	  },
	  {
		"name": "Coinex-BTC-USDT",
		"definition": {
		  "url": "https://api.coinex.com/v1/market/ticker?market=BTCUSDT",
		  "headers": { "Content-Type": "application/json" },
		  "method": "GET",
		  "reducers": [
			{ "function": "PARSE", "args": ["data", "ticker", "last"] },
			{ "function": "POW10", "args": 8 },
			{ "function": "ROUND" }
		  ]
		}
	  }
	]
  }`}

func setup() (*fiber.App, error) {
	mb := bus.New(10)

	app, err := utils.Setup(utils.SetupInfo{
		Version: "",
		Bus:     mb,
	})
	if err != nil {
		return nil, err
	}
	v1 := app.Group("/api/v1")
	adapter.Routes(v1)

	return app, nil
}

func insertSampleData(app *fiber.App, ctx context.Context) (func() error, error) {
	var insertData = make([]adapter.AdapterInsertModel, len(sampleData))
	var insertResults = make([]adapter.AdapterModel, len(sampleData))

	for i := range insertData {
		err := json.Unmarshal([]byte(sampleData[i]), &insertData[i])
		if err != nil {
			return nil, err
		}
	}

	for i := range insertResults {
		tmp, err := tests.PostRequest[adapter.AdapterModel](app, "/api/v1/adapter", insertData[i])
		if err != nil {
			return nil, err
		}
		insertResults[i] = tmp
	}

	return cleanupSampleData(app, ctx, insertResults), nil
}

func cleanupSampleData(app *fiber.App, ctx context.Context, insertResult []adapter.AdapterModel) func() error {
	return func() error {
		for i := range insertResult {
			_, err := tests.DeleteRequest[adapter.AdapterModel](app, "/api/v1/adapter/"+strconv.FormatInt(*insertResult[i].Id, 10), nil)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func TestMain(m *testing.M) {
	// setup
	code := m.Run()

	db.ClosePool()
	db.CloseRedis()

	// teardown
	os.Exit(code)
}
