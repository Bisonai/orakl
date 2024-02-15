package tests

import (
	"bisonai.com/orakl/api/adapter"
	"bisonai.com/orakl/api/aggregator"
	"bisonai.com/orakl/api/chain"
	"bisonai.com/orakl/api/feed"
	"bisonai.com/orakl/api/utils"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestAggregator(t *testing.T) {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Print("env file is not found, continueing without .env file")
	}
	insertChain := chain.ChainInsertModel{Name: "aggregator-test-chain"}

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

	customTrue := utils.CustomBool(true)
	aggregatorUpdateData := aggregator.WrappedUpdateModel{
		Data: aggregator.AggregatorUpdateModel{
			Active: &customTrue,
			Chain:  insertChain.Name,
		},
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
	feed.Routes(v1)

	// insert chain, adapter (setup)
	chainInsertResult, err := utils.PostRequest[chain.ChainModel](app, "/api/v1/chain", insertChain)
	assert.Nil(t, err)

	adapterInsertResult, err := utils.PostRequest[adapter.AdapterModel](app, "/api/v1/adapter", adapterInsertData)
	assert.Nil(t, err)

	// read all before insertion
	readAllResult, err := utils.GetRequest[[]aggregator.AggregatorResultModel](app, "/api/v1/aggregator?chain="+insertChain.Name, nil)
	assert.Nil(t, err)
	totalBefore := len(readAllResult)

	// insert
	aggregatorInsertResult, err := utils.PostRequest[aggregator.AggregatorResultModel](app, "/api/v1/aggregator", aggregatorInsertData)
	assert.Nil(t, err)

	// read all after insertion
	readAllAfter, err := utils.GetRequest[[]aggregator.AggregatorResultModel](app, "/api/v1/aggregator?chain="+insertChain.Name, nil)
	assert.Nil(t, err)
	totalAfter := len(readAllAfter)
	assert.Less(t, totalBefore, totalAfter)

	// read by hash and chain
	singleReadResult, err := utils.GetRequest[aggregator.AggregatorResultModel](app, "/api/v1/aggregator/"+aggregatorInsertData["aggregatorHash"].(string)+"/"+insertChain.Name, nil)
	assert.Nil(t, err)
	// FIXME: singleReadResult has more detailed info, should check differently
	assert.Equalf(t, aggregatorInsertResult.AggregatorId, singleReadResult.AggregatorId, "should read single element")

	// update by id
	patchResult, err := utils.PatchRequest[aggregator.AggregatorResultModel](app, "/api/v1/aggregator/"+aggregatorInsertResult.AggregatorHash, aggregatorUpdateData)
	assert.Nil(t, err)
	singleReadResult, err = utils.GetRequest[aggregator.AggregatorResultModel](app, "/api/v1/aggregator/"+aggregatorInsertData["aggregatorHash"].(string)+"/"+insertChain.Name, nil)
	assert.Nil(t, err)
	assert.Equalf(t, patchResult, singleReadResult, "should be patched")

	// hash
	hashTestInsertData, _ := utils.DeepCopyMap(aggregatorInsertData)
	hashTestInsertData["aggregatorHash"] = ""
	hashResult, err := utils.PostRequest[aggregator.AggregatorHashComputeInputModel](app, "/api/v1/aggregator/hash?verify=false", hashTestInsertData)
	assert.Nil(t, err)
	assert.Equalf(t, aggregatorInsertData["aggregatorHash"].(string), hashResult.AggregatorHash, "hash should be same")

	// delete by id
	deleteResult, err := utils.DeleteRequest[aggregator.AggregatorResultModel](app, "/api/v1/aggregator/"+aggregatorInsertResult.AggregatorId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, patchResult, deleteResult, "should be deleted")

	readAllResultAfterDeletion, err := utils.GetRequest[[]aggregator.AggregatorResultModel](app, "/api/v1/aggregator?chain="+insertChain.Name, nil)
	assert.Nil(t, err)
	assert.Less(t, len(readAllResultAfterDeletion), totalAfter)

	// cleanup
	feeds, _ := utils.GetRequest[[]feed.FeedModel](app, "/api/v1/feed", nil)
	for _, f := range feeds {
		_, err = utils.DeleteRequest[feed.FeedModel](app, "/api/v1/feed/"+f.FeedId.String(), nil)
		assert.Nil(t, err)
	}

	_, err = utils.DeleteRequest[adapter.AdapterModel](app, "/api/v1/adapter/"+adapterInsertResult.AdapterId.String(), nil)
	assert.Nil(t, err)
	_, err = utils.DeleteRequest[chain.ChainModel](app, "/api/v1/chain/"+chainInsertResult.ChainId.String(), nil)
	assert.Nil(t, err)
}
