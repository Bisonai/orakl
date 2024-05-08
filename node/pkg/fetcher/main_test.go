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
    "name": "BNB-USDT",
    "feeds": [
      {
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
      },
      {
        "name": "KlaySwap-oBNB-oUSDT",
        "definition": {
          "chainId": "8217",
          "address": "0x14afeda13bc2028cef34d3f45d1b4e3f44747b9a",
          "type": "UniswapPool",
          "token0Decimals": 18,
          "token1Decimals": 6
        }
      }
    ],
    "fetchInterval": 2000,
    "address": "0x63606c0B4b330338a99abf2EBC61DBA10489E9E1",
    "aggregateInterval": 5000,
    "submitInterval": 15000
  }`, `{
    "name": "BTC-USDT",
    "feeds": [
      {
        "name": "Bybit-BTC-USDT",
        "definition": {
          "url": "https://api.bybit.com/derivatives/v3/public/tickers?symbol=BTCUSDT",
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
        "name": "Binance-BTC-USDT",
        "definition": {
          "url": "https://api.binance.com/api/v3/avgPrice?symbol=BTCUSDT",
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
        "name": "Coinbase-BTC-USDT",
        "definition": {
          "url": "https://api.coinbase.com/v2/exchange-rates?currency=BTC",
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
        "name": "Kucoin-BTC-USDT",
        "definition": {
          "url": "https://api.kucoin.com/api/v1/market/orderbook/level1?symbol=BTC-USDT",
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
        "name": "Crypto-BTC-USDT",
        "definition": {
          "url": "https://api.crypto.com/v2/public/get-ticker?instrument_name=BTC_USDT",
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
        "name": "Btse-BTC-USDT",
        "definition": {
          "url": "https://api.btse.com/spot/api/v3.2/price?symbol=BTC-USDT",
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
        "name": "Gateio-BTC-USDT",
        "definition": {
          "url": "https://api.gateio.ws/api/v4/spot/tickers?currency_pair=BTC_USDT",
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
        "name": "Coinex-BTC-USDT",
        "definition": {
          "url": "https://api.coinex.com/v1/market/ticker?market=BTCUSDT",
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
    "address": "0x55faF24D49026be2bE81286783bD4605E0574bBd",
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
