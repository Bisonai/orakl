package fetcher

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/adapter"
	"bisonai.com/orakl/node/pkg/admin/tests"
	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

var sampleData = `{
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
}`

var insertResult adapter.AdapterModel

func setup() (*fiber.App, error) {
	app, err := utils.Setup("")
	if err != nil {
		return nil, err
	}
	v1 := app.Group("/api/v1")
	adapter.Routes(v1)

	return app, nil
}

func insertSampleData(app *fiber.App, ctx context.Context) error {
	var insertData adapter.AdapterInsertModel
	err := json.Unmarshal([]byte(sampleData), &insertData)
	if err != nil {
		return err
	}

	tmp, err := tests.PostRequest[adapter.AdapterModel](app, "/api/v1/adapter", insertData)
	if err != nil {
		return err
	}

	insertResult = tmp
	return nil
}

func cleanupSampleData(app *fiber.App, ctx context.Context) error {
	_, err := tests.DeleteRequest[adapter.AdapterModel](app, "/api/v1/adapter/"+strconv.FormatInt(*insertResult.Id, 10), nil)
	if err != nil {
		return err
	}
	return nil
}

func TestFetcherInitialize(t *testing.T) {
	admin, err := setup()
	if err != nil {
		t.Fatalf("error setting up admin: %v", err)
	}
	err = insertSampleData(admin, context.Background())
	if err != nil {
		t.Fatalf("error inserting sample data: %v", err)
	}
	defer admin.Shutdown()
	defer cleanupSampleData(admin, context.Background())
	b := bus.NewMessageBus()
	fetcher := NewFetcher(b)
	fetcher.initialize(context.Background())
	assert.Greater(t, len(fetcher.Adapters), 0)
	assert.Greater(t, len(fetcher.Adapters[0].Feeds), 0)
}

func TestFetcherFetch(t *testing.T) {
	admin, err := setup()
	if err != nil {
		t.Fatalf("error setting up admin: %v", err)
	}
	err = insertSampleData(admin, context.Background())
	if err != nil {
		t.Fatalf("error inserting sample data: %v", err)
	}
	defer admin.Shutdown()
	defer cleanupSampleData(admin, context.Background())

	b := bus.NewMessageBus()
	fetcher := NewFetcher(b)
	fetcher.initialize(context.Background())
	result, err := fetcher.fetch(fetcher.Adapters[0])
	if err != nil {
		t.Fatalf("error fetching: %v", err)
	}
	assert.Greater(t, len(result), 0)
}

func TestFetcherRunAdapter(t *testing.T) {
	admin, err := setup()
	if err != nil {
		t.Fatalf("error setting up admin: %v", err)
	}
	err = insertSampleData(admin, context.Background())
	if err != nil {
		t.Fatalf("error inserting sample data: %v", err)
	}
	defer admin.Shutdown()
	defer cleanupSampleData(admin, context.Background())

	b := bus.NewMessageBus()
	fetcher := NewFetcher(b)
	fetcher.initialize(context.Background())
	err = fetcher.runAdapter(context.Background())
	if err != nil {
		t.Fatalf("error running adapter: %v", err)
	}

	// read aggregate from db
	res, err := db.QueryRow[Aggregate](context.Background(), "SELECT * FROM local_aggregates WHERE name = @name", map[string]any{"name": fetcher.Adapters[0].Name})
	if err != nil {
		t.Fatalf("error reading from db: %v", err)
	}
	assert.NotNil(t, res)

	// cleanup aggregate from db
	_, err = db.Query(context.Background(), "DELETE FROM local_aggregates WHERE name = @name", map[string]any{"name": fetcher.Adapters[0].Name})
	if err != nil {
		t.Fatalf("error cleaning up from db: %v", err)
	}
}
