package tests

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"bisonai.com/orakl/api/adapter"
	"bisonai.com/orakl/api/aggregate"
	"bisonai.com/orakl/api/aggregator"
	"bisonai.com/orakl/api/chain"
	"bisonai.com/orakl/api/feed"
	"bisonai.com/orakl/api/utils"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

/*
sample data

POST localhost:3111/api/v1/chain

{
    "name":"aggregate-test-chain"
}

POST localhost:3111/api/v1/adapter

{
    "adapterHash": "0xbb555a249d01133784fa04c608ce03c129f73f2a1ef7473d0cfffdc4bcba794e",
	"name":         "BTC-USD",
	"decimals":     8,
    "feeds": [
        {
				"name": "Binance-BTC-USD-adapter",
				"definition": {
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
				}
			}
    ]
}

POST localhost:3111/api/v1/aggregator

{
    "aggregatorHash":    "0x9ca45583d7b9b061d9e8a20d6a874fcfba50c7a9dbc9c65c3792b4ef0b31e7b9",
		"active":             false,
		"name":               "BTC-USD",
		"address":            "0x222",
		"heartbeat":          10000,
		"threshold":          0.04,
		"absoluteThreshold": 0.1,
		"adapterHash":       "0xbb555a249d01133784fa04c608ce03c129f73f2a1ef7473d0cfffdc4bcba794e",
		"chain":              "aggregate-test-chain"
}
*/

func TestAggregate(t *testing.T) {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Print("env file is not found, continueing without .env file")
	}

	insertChain := chain.ChainInsertModel{Name: "aggregate-test-chain"}

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

	// aggregatorInsertData := aggregator.AggregatorInsertModel{
	// 	AGGREGATOR_HASH:    "0x9ca45583d7b9b061d9e8a20d6a874fcfba50c7a9dbc9c65c3792b4ef0b31e7b9",
	// 	ACTIVE:             false,
	// 	NAME:               "BTC-USD",
	// 	ADDRESS:            "0x222",
	// 	HEARTBEAT:          10000,
	// 	THRESHOLD:          0.04,
	// 	ABSOLUTE_THRESHOLD: 0.1,
	// 	ADAPTER_HASH:       _adapterInsertData.ADAPTER_HASH,
	// 	CHAIN:              insertChain.NAME,
	// 	FETCHER_TYPE:       0,
	// }

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
	aggregate.Routes(v1)
	feed.Routes(v1)

	// insert chain, adapter, and aggregator before test
	chainInsertResult, err := utils.PostRequest[chain.ChainModel](app, "/api/v1/chain", insertChain)
	assert.Nil(t, err)

	adapterInsertResult, err := utils.PostRequest[adapter.AdapterModel](app, "/api/v1/adapter", adapterInsertData)
	assert.Nil(t, err)

	aggregatorInsertResult, err := utils.PostRequest[aggregator.AggregatorResultModel](app, "/api/v1/aggregator", aggregatorInsertData)
	assert.Nil(t, err)

	// read all before insertion
	readAllResult, err := utils.GetRequest[[]aggregate.AggregateModel](app, "/api/v1/aggregate", nil)
	assert.Nil(t, err)
	totalBefore := len(readAllResult)

	now := time.Now().Truncate(time.Second)

	// insert
	insertValue := utils.CustomInt64(10)
	aggregateInsertData := aggregate.AggregateInsertModel{
		Timestamp:    &utils.CustomDateTime{Time: now},
		AggregatorId: aggregatorInsertResult.AggregatorId,
		Value:        &insertValue}
	wrappedAggregateInsertData := aggregate.WrappedInsertModel{Data: aggregateInsertData}

	aggregateInsertResult, err := utils.PostRequest[aggregate.AggregateModel](app, "/api/v1/aggregate", wrappedAggregateInsertData)
	assert.Nil(t, err)

	// read all after insertion
	readAllAfter, err := utils.GetRequest[[]aggregate.AggregateModel](app, "/api/v1/aggregate", nil)
	assert.Nil(t, err)
	totalAfter := len(readAllAfter)
	assert.Less(t, totalBefore, totalAfter)

	// read single
	singleReadResult, err := utils.GetRequest[aggregate.AggregateModel](app, "/api/v1/aggregate/"+aggregateInsertResult.AggregateId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, aggregateInsertResult, singleReadResult, "should get single element")

	// read latest by hash
	latestByHashResult, err := utils.GetRequest[aggregate.AggregateModel](app, "/api/v1/aggregate/hash/"+aggregatorInsertData["aggregatorHash"].(string)+"/latest", nil)
	assert.Nil(t, err)
	assert.Equalf(t, aggregateInsertResult, latestByHashResult, "should get latest by hash")

	// read latest by aggregatorId
	latestByAggregatorIdResult, err := utils.GetRequest[aggregate.AggregateRedisValueModel](app, "/api/v1/aggregate/id/"+aggregatorInsertResult.AggregatorId.String()+"/latest", nil)
	assert.Nil(t, err)
	assert.Equalf(t, aggregateInsertResult.Timestamp, latestByAggregatorIdResult.Timestamp, "should get latest by aggregatorId")
	assert.Equalf(t, aggregateInsertResult.Value, latestByAggregatorIdResult.Value, "should get latest by aggregatorId")

	// should update by id
	updateValue := utils.CustomInt64(20)
	aggregateUpdateData := aggregateInsertData
	aggregateUpdateData.Value = &updateValue
	updateResult, err := utils.PatchRequest[aggregate.AggregateModel](app, "/api/v1/aggregate/"+aggregateInsertResult.AggregateId.String(), aggregateUpdateData)
	assert.Nil(t, err)
	singleReadResult, err = utils.GetRequest[aggregate.AggregateModel](app, "/api/v1/aggregate/"+aggregateInsertResult.AggregateId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, updateResult, singleReadResult, "should get single element")

	// should delete by id
	deleteResult, err := utils.DeleteRequest[aggregate.AggregateModel](app, "/api/v1/aggregate/"+aggregateInsertResult.AggregateId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, updateResult, deleteResult, "should be deleted")
	readAllAfterDeletion, err := utils.GetRequest[[]aggregate.AggregateModel](app, "/api/v1/aggregate", nil)
	assert.Nil(t, err)
	assert.Less(t, len(readAllAfterDeletion), totalAfter)

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
