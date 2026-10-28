//nolint:all
package fetcher

import (
	"context"
	"encoding/json"
	"testing"

	"net/http"
	"net/http/httptest"

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
