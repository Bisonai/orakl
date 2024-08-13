//nolint:all
package fetcher

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"net/http"
	"net/http/httptest"

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

func TestSetFeedDataBuffer(t *testing.T) {
	ctx := context.Background()
	feedData := []*FeedData{
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

	result, err := db.LRangeObject[*FeedData](ctx, keys.FeedDataBufferKey(), 0, -1)
	if err != nil {
		t.Fatalf("error getting feed data buffer: %v", err)
	}

	assert.Equal(t, 2, len(result))
	assert.Contains(t, result, feedData[0])
	assert.Contains(t, result, feedData[1])
}

func TestGetFeedDataBuffer(t *testing.T) {
	ctx := context.Background()
	feedData := []*FeedData{
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
	feedData := []*FeedData{}

	for i, feed := range feeds {
		now := time.Now().Round(time.Second)
		feedData = append(feedData, &FeedData{
			FeedID:    int32(*feed.ID),
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
	assert.Contains(t, result, *feedData[0])
}
