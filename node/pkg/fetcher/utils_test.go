//nolint:all
package fetcher

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"net/http"
	"net/http/httptest"

	"bisonai.com/orakl/node/pkg/aggregator"
	"bisonai.com/orakl/node/pkg/common/keys"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestFetchSingle(t *testing.T) {
	ctx := context.Background()
	mockServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`{
			"retCode": 0,
			"retMsg": "OK",
			"result": {
			  "category": "",
			  "list": [
				{
				  "symbol": "ADAUSDT",
				  "bidPrice": "0.4675",
				  "askPrice": "0.4676",
				  "lastPrice": "0.4676",
				  "lastTickDirection": "ZeroPlusTick",
				  "prevPrice24h": "0.4885",
				  "price24hPcnt": "-0.042784",
				  "highPrice24h": "0.4890",
				  "lowPrice24h": "0.4435",
				  "prevPrice1h": "0.4675",
				  "markPrice": "0.4675",
				  "indexPrice": "0.4674",
				  "openInterest": "125794431",
				  "turnover24h": "121910399.6363",
				  "volume24h": "261804569.0000",
				  "fundingRate": "0.0001",
				  "nextFundingTime": "1716537600000",
				  "predictedDeliveryPrice": "",
				  "basisRate": "",
				  "deliveryFeeRate": "",
				  "deliveryTime": "0",
				  "openInterestValue": "58808896.49"
				}
			  ]
			},
			"retExtInfo": {

			},
			"time": 1716526378174
		  }`))
	}))
	defer mockServer.Close()

	rawDefinition := `
	{
        "url": "` + mockServer.URL + `",
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
	}`

	definition := new(Definition)
	err := json.Unmarshal([]byte(rawDefinition), &definition)
	if err != nil {
		t.Fatalf("error unmarshalling definition: %v", err)
	}

	result, err := FetchSingle(ctx, definition)
	if err != nil {
		t.Fatalf("error fetching single: %v", err)
	}
	assert.Greater(t, result, float64(0))
}

func TestGetTokenPrice(t *testing.T) {
	rawDefinition := `{
        "chainId": "1",
        "address": "0x9db9e0e53058c89e5b94e29621a205198648425b",
        "type": "UniswapPool",
        "token0Decimals": 8,
        "token1Decimals": 6
	}`

	definition := new(Definition)
	err := json.Unmarshal([]byte(rawDefinition), &definition)
	if err != nil {
		t.Fatalf("error unmarshalling definition: %v", err)
	}

	sqrtPriceX96 := new(big.Int)
	sqrtPriceX96.SetString("2055909007346292057510600778491", 10)
	result, err := getTokenPrice(sqrtPriceX96, definition)
	if err != nil {
		t.Fatalf("error getting token price: %v", err)
	}

	assert.Equal(t, float64(6.733620107923e+12), result)
}

func TestSetLatestFeedData(t *testing.T) {
	ctx := context.Background()
	feedData := []FeedData{
		{
			FeedID: 1,
			Value:  0.1,
		},
		{
			FeedID: 2,
			Value:  0.2,
		},
	}

	err := setLatestFeedData(ctx, feedData, 1*time.Second)
	if err != nil {
		t.Fatalf("error setting latest feed data: %v", err)
	}
	keys := []string{keys.LatestFeedDataKey(1), keys.LatestFeedDataKey(2)}

	defer db.Del(ctx, keys[0])
	defer db.Del(ctx, keys[1])

	result, err := db.MGetObject[FeedData](ctx, keys)
	if err != nil {
		t.Fatalf("error getting latest feed data: %v", err)
	}

	assert.Equal(t, 2, len(result))
	assert.Contains(t, result, feedData[0])
	assert.Contains(t, result, feedData[1])
}

func TestGetLatestFeedData(t *testing.T) {
	ctx := context.Background()
	feedData := []FeedData{
		{
			FeedID: 1,
			Value:  0.1,
		},
		{
			FeedID: 2,
			Value:  0.2,
		},
	}

	keys := []string{keys.LatestFeedDataKey(1), keys.LatestFeedDataKey(2)}
	err := setLatestFeedData(ctx, feedData, 1*time.Second)
	if err != nil {
		t.Fatalf("error setting latest feed data: %v", err)
	}
	defer db.Del(ctx, keys[0])
	defer db.Del(ctx, keys[1])

	result, err := getLatestFeedData(ctx, []int32{1, 2})
	if err != nil {
		t.Fatalf("error getting latest feed data: %v", err)
	}

	assert.Equal(t, 2, len(result))
	assert.Contains(t, result, feedData[0])
	assert.Contains(t, result, feedData[1])
}

func TestSetFeedDataBuffer(t *testing.T) {
	ctx := context.Background()
	feedData := []FeedData{
		{
			FeedID: 1,
			Value:  0.1,
		},
		{
			FeedID: 2,
			Value:  0.2,
		},
	}

	err := setFeedDataBuffer(ctx, feedData)
	if err != nil {
		t.Fatalf("error setting feed data buffer: %v", err)
	}

	defer db.Del(ctx, keys.FeedDataBufferKey())

	result, err := db.LRangeObject[FeedData](ctx, keys.FeedDataBufferKey(), 0, -1)
	if err != nil {
		t.Fatalf("error getting feed data buffer: %v", err)
	}

	assert.Equal(t, 2, len(result))
	assert.Contains(t, result, feedData[0])
	assert.Contains(t, result, feedData[1])
}

func TestGetFeedDataBuffer(t *testing.T) {
	ctx := context.Background()
	feedData := []FeedData{
		{
			FeedID: 1,
			Value:  0.1,
		},
		{
			FeedID: 2,
			Value:  0.2,
		},
	}

	err := setFeedDataBuffer(ctx, feedData)
	if err != nil {
		t.Fatalf("error setting feed data buffer: %v", err)
	}

	defer db.Del(ctx, keys.FeedDataBufferKey())

	result, err := getFeedDataBuffer(ctx)
	if err != nil {
		t.Fatalf("error getting feed data buffer: %v", err)
	}

	assert.Equal(t, 2, len(result))
	assert.Contains(t, result, feedData[0])
	assert.Contains(t, result, feedData[1])
}

func TestInsertLocalAggregatePgsql(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := clean(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	configs := testItems.insertedConfigs
	for i, config := range configs {
		insertLocalAggregateErr := insertLocalAggregatePgsql(ctx, config.Id, float64(i)+5)
		if insertLocalAggregateErr != nil {
			t.Fatalf("error inserting local aggregate pgsql: %v", insertLocalAggregateErr)
		}
	}

	defer db.QueryWithoutResult(ctx, "DELETE FROM local_aggregates", nil)
	result, err := db.QueryRows[aggregator.LocalAggregate](ctx, "SELECT * FROM local_aggregates", nil)
	if err != nil {
		t.Fatalf("error getting local aggregate pgsql: %v", err)
	}

	assert.Equal(t, len(configs), len(result))
}

func TestInsertLocalAggregateRdb(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := clean(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	configs := testItems.insertedConfigs
	for i, config := range configs {
		err := insertLocalAggregateRdb(ctx, config.Id, float64(i)+5)
		if err != nil {
			t.Fatalf("error inserting local aggregate rdb: %v", err)
		}
		defer db.Del(ctx, keys.LocalAggregateKey(config.Id))
	}

	for _, config := range configs {
		result, err := db.GetObject[aggregator.LocalAggregate](ctx, keys.LocalAggregateKey(config.Id))
		if err != nil {
			t.Fatalf("error getting local aggregate rdb: %v", err)
		}
		assert.Equal(t, int32(config.Id), result.ConfigID)
	}
}

func TestCopyFeedData(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := clean(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	feeds := testItems.insertedFeeds
	feedData := []FeedData{}

	for i, feed := range feeds {
		now := time.Now().Round(time.Second)
		feedData = append(feedData, FeedData{
			FeedID:    int32(*feed.Id),
			Value:     float64(i) + 5,
			Timestamp: &now,
		})
	}

	err = setFeedDataBuffer(ctx, feedData)
	if err != nil {
		t.Fatalf("error setting feed data buffer: %v", err)
	}

	defer db.Del(ctx, keys.FeedDataBufferKey())

	err = copyFeedData(ctx, feedData)
	if err != nil {
		t.Fatalf("error copying feed data: %v", err)
	}

	defer db.QueryWithoutResult(ctx, "DELETE FROM feed_data", nil)
	result, err := db.QueryRows[FeedData](ctx, "SELECT * FROM feed_data", nil)
	if err != nil {
		t.Fatalf("error getting feed data: %v", err)
	}
	assert.Contains(t, result, feedData[0])
}
