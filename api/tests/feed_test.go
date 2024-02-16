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

func containsFeed(arr []feed.FeedModel, target feed.FeedModel) bool {
	for _, f := range arr {
		if f.FeedId == target.FeedId {
			return true
		}
	}
	return false
}

func TestFeed(t *testing.T) {
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

	// read all before insert
	readAllResult, err := utils.GetRequest[[]feed.FeedModel](app, "/api/v1/feed", nil)
	assert.Nil(t, err)
	totalBefore := len(readAllResult)

	// insert adapter which adds feed
	adapterInsertResult, err := utils.PostRequest[adapter.AdapterModel](app, "/api/v1/adapter", adapterInsertData)
	assert.Nil(t, err)

	// read all after insertion
	readAllResultAfter, err := utils.GetRequest[[]feed.FeedModel](app, "/api/v1/feed", nil)
	assert.Nil(t, err)
	totalAfter := len(readAllResultAfter)
	assert.Less(t, totalBefore, totalAfter)

	// read single
	lastElement := readAllResultAfter[len(readAllResultAfter)-1]
	singleReadResult, err := utils.GetRequest[feed.FeedModel](app, "/api/v1/feed/"+lastElement.FeedId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, lastElement, singleReadResult, "should get single element")

	// delete added feeds
	for _, f := range readAllResultAfter {
		if !containsFeed(readAllResult, f) {
			deletedFeed, err := utils.DeleteRequest[feed.FeedModel](app, "/api/v1/feed/"+f.FeedId.String(), nil)
			assert.Nil(t, err)
			assert.NotNil(t, deletedFeed)
		}
	}

	// delete adapter (cleanup)
	_, err = utils.DeleteRequest[adapter.AdapterModel](app, "/api/v1/adapter/"+adapterInsertResult.AdapterId.String(), nil)
	assert.Nil(t, err)

	// read all after deletion and cleanup
	readAllResultAfterDeletion, err := utils.GetRequest[[]feed.FeedModel](app, "/api/v1/feed", nil)
	assert.Nil(t, err)
	assert.Less(t, len(readAllResultAfterDeletion), totalAfter)
}
