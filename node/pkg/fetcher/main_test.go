//nolint:all
package fetcher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/config"
	"bisonai.com/orakl/node/pkg/admin/feed"
	"bisonai.com/orakl/node/pkg/admin/fetcher"
	"bisonai.com/orakl/node/pkg/admin/proxy"
	"bisonai.com/orakl/node/pkg/admin/tests"
	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

const (
	mockReply0 = `{
    "mins": 5,
    "price": "1.01827085",
    "closeTime": 1597204784937
  }`

	mockReply1 = `{
    "retCode": 0,
    "retMsg": "OK",
    "result": {
      "category": "",
      "list": [
        {
          "symbol": "DOGEUSDT",
          "bidPrice": "0.15845",
          "askPrice": "0.15846",
          "lastPrice": "0.15845",
          "lastTickDirection": "ZeroMinusTick",
          "prevPrice24h": "0.16706",
          "price24hPcnt": "-0.051538",
          "highPrice24h": "0.17140",
          "lowPrice24h": "0.15147",
          "prevPrice1h": "0.15671",
          "markPrice": "0.15846",
          "indexPrice": "0.15839",
          "openInterest": "1522687772",
          "turnover24h": "773341302.2606",
          "volume24h": "4789388852.0000",
          "fundingRate": "0.0001",
          "nextFundingTime": "1716537600000",
          "predictedDeliveryPrice": "",
          "basisRate": "",
          "deliveryFeeRate": "",
          "deliveryTime": "0",
          "openInterestValue": "241285104.35"
        }
      ]
    },
    "retExtInfo": {

    },
    "time": 1716537538486
  }`
)

type TestItems struct {
	admin           *fiber.App
	messageBus      *bus.MessageBus
	app             *App
	insertedConfigs []config.ConfigModel
	insertedFeeds   []feed.FeedModel
	mockDataSource  []*httptest.Server
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
	feed.Routes(v1)

	app := New(mb)

	mockDataSource1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(mockReply0))
	}))

	mockDataSource2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(mockReply1))
	}))

	testItems.mockDataSource = []*httptest.Server{mockDataSource1, mockDataSource2}
	testItems.admin = admin
	testItems.messageBus = mb
	testItems.app = app

	configs, feeds, err := insertSampleData(ctx, testItems)
	if err != nil {
		return nil, nil, err
	}
	testItems.insertedConfigs = configs
	testItems.insertedFeeds = feeds

	return cleanup(ctx, testItems), testItems, nil
}

func insertSampleData(ctx context.Context, testItems *TestItems) ([]config.ConfigModel, []feed.FeedModel, error) {
	cleanupErr := db.QueryWithoutResult(ctx, "DELETE FROM configs", nil)
	if cleanupErr != nil {
		return nil, nil, cleanupErr
	}

	var sampleData = []string{`{
    "name": "DAI-USDT",
    "feeds": [
      {
        "name": "Binance-DAI-USDT",
        "definition": {
          "url": "` + testItems.mockDataSource[0].URL + `",
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
    "aggregateInterval": 5000,
    "submitInterval": 15000
  }`, `{
  "name": "DOGE-USDT",
  "feeds": [
    {
      "name": "Bybit-DOGE-USDT",
      "definition": {
        "url": "` + testItems.mockDataSource[1].URL + `",
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
    }
  ],
  "fetchInterval": 2000,
  "aggregateInterval": 5000,
  "submitInterval": 15000
  }`}

	var insertData = make([]config.ConfigInsertModel, len(sampleData))
	var insertResults = make([]config.ConfigModel, len(sampleData))

	for i := range insertData {
		err := json.Unmarshal([]byte(sampleData[i]), &insertData[i])
		if err != nil {
			return nil, nil, err
		}
	}

	for i := range insertResults {
		tmp, err := tests.PostRequest[config.ConfigModel](testItems.admin, "/api/v1/config", insertData[i])
		if err != nil {
			return nil, nil, err
		}
		insertResults[i] = tmp
	}

	insertedFeeds, err := tests.GetRequest[[]feed.FeedModel](testItems.admin, "/api/v1/feed", nil)
	if err != nil {
		return nil, nil, err
	}

	return insertResults, insertedFeeds, nil
}

func cleanup(ctx context.Context, testItems *TestItems) func() error {
	return func() error {
		if err := testItems.admin.Shutdown(); err != nil {
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
		err = testItems.app.stopAllFetchers(ctx)
		if err != nil {
			return err
		}

		for _, server := range testItems.mockDataSource {
			server.Close()
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
