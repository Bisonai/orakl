package fetcher

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/config"
	"bisonai.com/orakl/node/pkg/admin/fetcher"
	"bisonai.com/orakl/node/pkg/admin/proxy"
	"bisonai.com/orakl/node/pkg/admin/tests"
	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

var sampleData = []string{`{
  "name": "DAI-USDT",
  "feeds": [
    {
      "name": "Binance-DAI-USDT",
      "definition": {
        "url": "https://api.binance.com/api/v3/avgPrice?symbol=DAIUSDT",
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
      "name": "Crypto-DAI-USDT",
      "definition": {
        "url": "https://api.crypto.com/v2/public/get-ticker?instrument_name=DAI_USDT",
        "headers": {
          "Content-Type": "application/json"
        },
        "method": "GET",
        "reducers": [
          {
            "function": "PARSE",
            "args": [
              "result",
              "data"
            ]
          },
          {
            "function": "INDEX",
            "args": 0
          },
          {
            "function": "PARSE",
            "args": [
              "a"
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
      "name": "Coinbase-DAI-USDT",
      "definition": {
        "url": "https://api.coinbase.com/v2/exchange-rates?currency=DAI",
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
      "name": "Gateio-DAI-USDT",
      "definition": {
        "url": "https://api.gateio.ws/api/v4/spot/tickers?currency_pair=DAI_USDT",
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
      "name": "Coinex-DAI-USDT",
      "definition": {
        "url": "https://api.coinex.com/v1/market/ticker?market=DAIUSDT",
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
    },
    {
      "name": "UniswapV3-DAI-USDT",
      "definition": {
        "chainId": "1",
        "address": "0x48da0965ab2d2cbf1c17c09cfb5cbe67ad5b1406",
        "type": "UniswapPool",
        "token0Decimals": 18,
        "token1Decimals": 6
      }
    }
  ],
  "fetchInterval": 2000,
  "address": "0xc22Cd928deFce14292986aCE75a1BE1bcF100697",
  "aggregateInterval": 5000,
  "submitInterval": 15000
}`, `
"name": "DOGE-USDT",
"feeds": [
  {
    "name": "Bybit-DOGE-USDT",
    "definition": {
      "url": "https://api.bybit.com/derivatives/v3/public/tickers?symbol=DOGEUSDT",
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
    "name": "Binance-DOGE-USDT",
    "definition": {
      "url": "https://api.binance.com/api/v3/avgPrice?symbol=DOGEUSDT",
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
    "name": "Kucoin-DOGE-USDT",
    "definition": {
      "url": "https://api.kucoin.com/api/v1/market/orderbook/level1?symbol=DOGE-USDT",
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
    "name": "Crypto-DOGE-USDT",
    "definition": {
      "url": "https://api.crypto.com/v2/public/get-ticker?instrument_name=DOGE_USDT",
      "headers": {
        "Content-Type": "application/json"
      },
      "method": "GET",
      "reducers": [
        {
          "function": "PARSE",
          "args": [
            "result",
            "data"
          ]
        },
        {
          "function": "INDEX",
          "args": 0
        },
        {
          "function": "PARSE",
          "args": [
            "a"
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
    "name": "Btse-DOGE-USDT",
    "definition": {
      "url": "https://api.btse.com/spot/api/v3.2/price?symbol=DOGE-USDT",
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
    "name": "Coinbase-DOGE-USDT",
    "definition": {
      "url": "https://api.coinbase.com/v2/exchange-rates?currency=DOGE",
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
    "name": "Gateio-DOGE-USDT",
    "definition": {
      "url": "https://api.gateio.ws/api/v4/spot/tickers?currency_pair=DOGE_USDT",
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
    "name": "Coinex-DOGE-USDT",
    "definition": {
      "url": "https://api.coinex.com/v1/market/ticker?market=DOGEUSDT",
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
  }
],
"fetchInterval": 2000,
"address": "0x4470866FE5265841a56795AA28842d0e9f04E770",
"aggregateInterval": 5000,
"submitInterval": 15000
}`}

type TestItems struct {
	admin      *fiber.App
	messageBus *bus.MessageBus
	app        *App
}

func setup(ctx context.Context) (func() error, *TestItems, error) {
	var testItems = new(TestItems)
	mb := bus.New(10)

	admin, err := utils.Setup(utils.SetupInfo{
		Version: "",
		Bus:     mb,
	})
	if err != nil {
		return nil, nil, err
	}
	v1 := admin.Group("/api/v1")
	config.Routes(v1)
	proxy.Routes(v1)
	fetcher.Routes(v1)

	app := New(mb)

	testItems.admin = admin
	testItems.messageBus = mb
	testItems.app = app

	err = insertSampleData(ctx, admin)
	if err != nil {
		return nil, nil, err
	}

	return cleanup(ctx, admin, app), testItems, nil
}

func insertSampleData(ctx context.Context, app *fiber.App) error {
	var insertData = make([]config.ConfigInsertModel, len(sampleData))
	var insertResults = make([]config.ConfigModel, len(sampleData))

	for i := range insertData {
		err := json.Unmarshal([]byte(sampleData[i]), &insertData[i])
		if err != nil {
			return err
		}
	}

	for i := range insertResults {
		tmp, err := tests.PostRequest[config.ConfigModel](app, "/api/v1/config", insertData[i])
		if err != nil {
			return err
		}
		insertResults[i] = tmp
	}

	return nil
}

func cleanup(ctx context.Context, admin *fiber.App, app *App) func() error {
	return func() error {
		if err := admin.Shutdown(); err != nil {
			return err
		}
		err := db.QueryWithoutResult(ctx, "DELETE FROM configs", nil)
		if err != nil {
			return err
		}

		err = db.QueryWithoutResult(ctx, "DELETE FROM local_aggregates", nil)
		if err != nil {
			return err
		}
		err = db.QueryWithoutResult(ctx, "DELETE FROM feeds", nil)
		if err != nil {
			return err
		}
		err = db.QueryWithoutResult(ctx, "DELETE FROM feed_data", nil)
		if err != nil {
			return err
		}
		err = app.stopAllFetchers(ctx)
		if err != nil {
			return err
		}
		return nil
	}
}

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	// setup
	code := m.Run()

	db.ClosePool()
	db.CloseRedis()

	// teardown
	os.Exit(code)
}
