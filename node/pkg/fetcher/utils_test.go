//nolint:all

package fetcher

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"net/http"
	"net/http/httptest"

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

	err := setLatestFeedData(ctx, feedData)
	if err != nil {
		t.Fatalf("error setting latest feed data: %v", err)
	}
	keys := []string{"latestFeedData:1", "latestFeedData:2"}

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

	keys := []string{"latestFeedData:1", "latestFeedData:2"}
	// err := db.MSetObject(ctx, map[string]any{
	// 	keys[0]: feedData[0],
	// 	keys[1]: feedData[1],
	// })
	// if err != nil {
	// 	t.Fatalf("error setting latest feed data: %v", err)
	// }
	err := setLatestFeedData(ctx, feedData)
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

	defer db.Del(ctx, "feedDataBuffer")

	result, err := db.LRangeObject[FeedData](ctx, "feedDataBuffer", 0, -1)
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

	defer db.Del(ctx, "feedDataBuffer")

	result, err := getFeedDataBuffer(ctx)
	if err != nil {
		t.Fatalf("error getting feed data buffer: %v", err)
	}

	assert.Equal(t, 2, len(result))
	assert.Contains(t, result, feedData[0])
	assert.Contains(t, result, feedData[1])
}
