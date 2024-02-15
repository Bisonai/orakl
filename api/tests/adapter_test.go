package tests

import (
	"bisonai.com/orakl/api/adapter"
	"bisonai.com/orakl/api/feed"
	"bisonai.com/orakl/api/utils"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestAdapter(t *testing.T) {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Print("env file is not found, continueing without .env file")
	}

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

	appConfig, _ := utils.Setup()

	pgxClient := appConfig.Postgres
	redisClient := appConfig.Redis
	app := appConfig.App

	defer pgxClient.Close()
	defer redisClient.Close()
	v1 := app.Group("/api/v1")
	adapter.Routes(v1)
	feed.Routes(v1)

	// read all before insertion
	readAllResult, err := utils.GetRequest[[]adapter.AdapterModel](app, "/api/v1/adapter", nil)
	assert.Nil(t, err)
	totalBefore := len(readAllResult)

	// insert
	adapterInsertResult, err := utils.PostRequest[adapter.AdapterModel](app, "/api/v1/adapter", adapterInsertData)
	assert.Nil(t, err)

	// read all after insertion
	readAllAfter, err := utils.GetRequest[[]adapter.AdapterModel](app, "/api/v1/adapter", nil)
	assert.Nil(t, err)
	totalAfter := len(readAllAfter)
	assert.Less(t, totalBefore, totalAfter)

	// get single
	singleReadResult, err := utils.GetRequest[adapter.AdapterModel](app, "/api/v1/adapter/"+adapterInsertResult.AdapterId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, adapterInsertResult, singleReadResult, "should read single element")

	// hash
	hashTestInsertData := adapterInsertData
	hashTestInsertData.AdapterHash = ""
	hashResult, err := utils.PostRequest[adapter.AdapterInsertModel](app, "/api/v1/adapter/hash?verify=false", hashTestInsertData)
	assert.Nil(t, err)
	assert.Equal(t, adapterInsertData.AdapterHash, hashResult.AdapterHash, "hash should be same")

	// delete by id
	feeds, _ := utils.GetRequest[[]feed.FeedModel](app, "/api/v1/feed", nil)

	for _, f := range feeds {
		_, err = utils.DeleteRequest[feed.FeedModel](app, "/api/v1/feed/"+f.FeedId.String(), nil)
		assert.Nil(t, err)
	}

	deleteResult, err := utils.DeleteRequest[adapter.AdapterModel](app, "/api/v1/adapter/"+adapterInsertResult.AdapterId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, singleReadResult, deleteResult, "should be deleted")

	readAllResultAfterDeletion, err := utils.GetRequest[[]adapter.AdapterModel](app, "/api/v1/adapter", nil)
	assert.Nil(t, err)
	assert.Less(t, len(readAllResultAfterDeletion), totalAfter)

}
