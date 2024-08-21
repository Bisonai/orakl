package main

import (
	"encoding/json"

	"bisonai.com/miko/node/pkg/utils/reducer"
	"bisonai.com/miko/node/pkg/utils/request"
	"github.com/rs/zerolog/log"
)

type Definition struct {
	Url      *string           `json:"url"`
	Headers  map[string]string `json:"headers"`
	Method   *string           `json:"method"`
	Reducers []reducer.Reducer `json:"reducers"`
	Location *string           `json:"location"`

	// dex specific
	Type           *string `json:"type"`
	ChainID        *string `json:"chainId"`
	Address        *string `json:"address"`
	Token0Decimals *int64  `json:"token0Decimals"`
	Token1Decimals *int64  `json:"token1Decimals"`
	Reciprocal     *bool   `json:"reciprocal"`
}

func fetch(definition *Definition) (float64, error) {
	rawResult, err := request.Request[interface{}](request.WithEndpoint(*definition.Url), request.WithHeaders(definition.Headers))
	if err != nil {
		return 0, err
	}

	return reducer.Reduce(rawResult, definition.Reducers)
}

const testDefStr = `{
        "url": "http://m.stock.naver.com/front-api/marketIndex/productDetail?category=exchange&reutersCode=FX_USDKRW",
        "headers": {
          "Content-Type": "application/json"
        },
        "method": "GET",
        "reducers": [
          {
            "function": "PARSE",
            "args": ["result", "closePrice"]
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
        ]
      }`

func main() {
	var def Definition
	err := json.Unmarshal([]byte(testDefStr), &def)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse definition")
		return
	}

	res, err := fetch(&def)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch")
		return
	}
	log.Info().Float64("Result", res).Msg("fetched")
}
