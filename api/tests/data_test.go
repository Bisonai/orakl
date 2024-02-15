package tests

import (
	"bisonai.com/orakl/api/adapter"
	"bisonai.com/orakl/api/aggregator"
	"bisonai.com/orakl/api/chain"
	"bisonai.com/orakl/api/data"
	"bisonai.com/orakl/api/feed"
	"bisonai.com/orakl/api/utils"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestData(t *testing.T) {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Print("env file is not found, continueing without .env file")
	}

	insertChain := chain.ChainInsertModel{Name: "data-test-chain"}

	_adapterInsertData := _AdapterInsertModel{
		AdapterHash: "0xbb555a249d01133784fa04c608ce03c129f73f2a1ef7473d0cfffdc4bcba794e",
		Name:        "BTC-USD",
		Decimals:    8,
	}

	adapterInsertData := AdapterInsertModel{
		_AdapterInsertModel: _adapterInsertData,
		Feeds: []feed.FeedInsertModel{
			{
				Name: "Binance-BTC-USD-adapter",
				Definition: json.RawMessage(`{
					"url": "https://api.binance.us/api/v3/ticker/price?symbol=BTCUSD",
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
				}`),
			},
		},
	}

	aggregatorInsertData := map[string]interface{}{
		"aggregatorHash":    "0x9ca45583d7b9b061d9e8a20d6a874fcfba50c7a9dbc9c65c3792b4ef0b31e7b9",
		"active":            false,
		"name":              "BTC-USD",
		"address":           "0x222",
		"heartbeat":         10000,
		"threshold":         0.04,
		"absoluteThreshold": 0.1,
		"adapterHash":       _adapterInsertData.AdapterHash,
		"chain":             insertChain.Name,
		"fetcherType":       0,
	}

	appConfig, _ := utils.Setup()

	pgxClient := appConfig.Postgres
	redisClient := appConfig.Redis
	app := appConfig.App

	defer pgxClient.Close()
	defer redisClient.Close()
	v1 := app.Group("/api/v1")

	chain.Routes(v1)
	adapter.Routes(v1)
	aggregator.Routes(v1)
	data.Routes(v1)
	feed.Routes(v1)

	// insert chain, adapter, and aggregator before test
	chainInsertResult, err := utils.PostRequest[chain.ChainModel](app, "/api/v1/chain", insertChain)
	assert.Nil(t, err)

	adapterInsertResult, err := utils.PostRequest[adapter.AdapterModel](app, "/api/v1/adapter", adapterInsertData)
	assert.Nil(t, err)

	aggregatorInsertResult, err := utils.PostRequest[aggregator.AggregatorResultModel](app, "/api/v1/aggregator", aggregatorInsertData)
	assert.Nil(t, err)

	insertedFeeds, err := utils.GetRequest[[]feed.FeedModel](app, "/api/v1/feed/adapter/"+adapterInsertResult.AdapterId.String(), nil)
	assert.Nil(t, err)

	// read all before insertion
	readAllResult, err := utils.GetRequest[[]data.DataResultModel](app, "/api/v1/data", nil)
	assert.Nil(t, err)
	totalBefore := len(readAllResult)

	insertData := map[string]interface{}{
		"data": []map[string]interface{}{
			{
				"aggregatorId": aggregatorInsertResult.AggregatorId,
				"timestamp":    time.Now().UTC().Format(utils.RFC3339Milli),
				"value":        2241772466578,
				"feedId":       insertedFeeds[0].FeedId,
			},
		},
	}

	count, err := utils.PostRequest[struct {
		COUNT int `json:"count"`
	}](app, "/api/v1/data", insertData)
	assert.Nil(t, err)
	assert.Equalf(t, 1, count.COUNT, "1 insert")

	// read all after insertion
	readAllResultAfter, err := utils.GetRequest[[]data.DataResultModel](app, "/api/v1/data", nil)
	assert.Nil(t, err)
	totalAfter := len(readAllResultAfter)
	assert.Less(t, totalBefore, totalAfter)

	// read single
	lastDataElement := readAllResultAfter[len(readAllResultAfter)-1]
	singleReadResult, err := utils.GetRequest[data.DataResultModel](app, "/api/v1/data/"+lastDataElement.DataId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, lastDataElement, singleReadResult, "should get single element")

	// delete by id
	insertedData, err := utils.GetRequest[[]data.DataResultModel](app, "/api/v1/data/feed/"+insertedFeeds[0].FeedId.String(), nil)
	assert.Nil(t, err)
	deletedData, err := utils.DeleteRequest[data.DataResultModel](app, "/api/v1/data/"+insertedData[0].DataId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, insertedData[0], deletedData, "should delete by id")

	// clean up

	feeds, _ := utils.GetRequest[[]feed.FeedModel](app, "/api/v1/feed", nil)
	for _, f := range feeds {
		_, err = utils.DeleteRequest[feed.FeedModel](app, "/api/v1/feed/"+f.FeedId.String(), nil)
		assert.Nil(t, err)
	}

	_, err = utils.DeleteRequest[aggregator.AggregatorResultModel](app, "/api/v1/aggregator/"+aggregatorInsertResult.AggregatorId.String(), nil)
	assert.Nil(t, err)
	_, err = utils.DeleteRequest[adapter.AdapterModel](app, "/api/v1/adapter/"+adapterInsertResult.AdapterId.String(), nil)
	assert.Nil(t, err)
	_, err = utils.DeleteRequest[chain.ChainModel](app, "/api/v1/chain/"+chainInsertResult.ChainId.String(), nil)
	assert.Nil(t, err)

}
