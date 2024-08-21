package tests

import (
	"encoding/json"
	"testing"

	"bisonai.com/miko/node/pkg/utils/reducer"
	"github.com/stretchr/testify/assert"
)

var reducers = `[
	{
	  "function": "INDEX",
	  "args": 0
	},
	{
	  "function": "PARSE",
	  "args": [
		"basePrice"
	  ]
	},
	{
	  "function": "DIVFROM",
	  "args": 1
	},
	{
	  "function": "POW10",
	  "args": 8
	},
	{
	  "function": "ROUND"
	}
  ]`
var sampleResult = `[
	{
	  "code": "FRX.KRWUSD",
	  "currencyCode": "USD",
	  "currencyName": "달러",
	  "country": "미국",
	  "name": "미국 (USD/KRW)",
	  "date": "2024-02-20",
	  "time": "15:31:14",
	  "recurrenceCount": 194,
	  "basePrice": 1338.00,
	  "openingPrice": 1333.80,
	  "highPrice": 1339.10,
	  "lowPrice": 1333.80,
	  "change": "RISE",
	  "changePrice": 2.50,
	  "cashBuyingPrice": 1361.41,
	  "cashSellingPrice": 1314.59,
	  "ttBuyingPrice": 1324.90,
	  "ttSellingPrice": 1351.10,
	  "tcBuyingPrice": null,
	  "fcSellingPrice": null,
	  "exchangeCommission": 7.1659,
	  "usDollarRate": 1.0000,
	  "high52wPrice": 1363.50,
	  "high52wDate": "2023-10-04",
	  "low52wPrice": 1257.50,
	  "low52wDate": "2023-07-18",
	  "currencyUnit": 1,
	  "provider": "하나은행",
	  "timestamp": 1708410689000,
	  "id": 79,
	  "modifiedAt": "2024-02-20T06:31:29.000+00:00",
	  "createdAt": "2016-10-21T06:13:34.000+00:00",
	  "signedChangePrice": 2.50,
	  "signedChangeRate": 0.0018719581,
	  "changeRate": 0.0018719581
	}
  ]`

func TestReduceAll(t *testing.T) {
	var red []reducer.Reducer
	err := json.Unmarshal([]byte(reducers), &red)
	if err != nil {
		t.Fatalf("error unmarshalling sample def: %v", err)
	}

	var res interface{}
	err = json.Unmarshal([]byte(sampleResult), &res)
	if err != nil {
		t.Fatalf("error unmarshalling sample result: %v", err)
	}

	result, err := reducer.Reduce(res, red)
	if err != nil {
		t.Fatalf("error reducing: %v", err)
	}
	assert.NotEqual(t, result, 0)
}
