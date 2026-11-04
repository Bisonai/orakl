package fetcher

import (
	"context"
	"strings"
	"time"

	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/utils/calculator"
	"bisonai.com/miko/node/pkg/utils/reducer"
	"bisonai.com/miko/node/pkg/utils/request"
	"github.com/rs/zerolog/log"
)

func FetchSingle(ctx context.Context, definition *Definition, reqOpts ...request.RequestOption) (float64, error) {
	requestOptions := []request.RequestOption{
		request.WithEndpoint(*definition.Url),
		request.WithHeaders(definition.Headers),
		request.WithTimeout(10 * time.Second),
	}

	requestOptions = append(requestOptions, reqOpts...)

	rawResult, err := request.Request[interface{}](requestOptions...)
	if err != nil {
		return 0, err
	}

	return reducer.Reduce(rawResult, definition.Reducers)
}

func calculateVWAP(feedData []*FeedData) (float64, error) {
	if len(feedData) == 0 {
		log.Debug().Str("Player", "Fetcher").Msg("no feed data to calculate VWAP")
		return 0, nil
	}

	totalPrice := 0.0
	totalVolume := 0.0
	for _, data := range feedData {
		totalPrice += data.Value * data.Volume
		totalVolume += data.Volume
	}

	if totalVolume == 0 {
		log.Debug().Str("Player", "Fetcher").Msg("total volume is zero to calculate VWAP")
		return 0, errorSentinel.ErrLocalAggregatorZeroVolume
	}

	return totalPrice / totalVolume, nil
}

func calculateMedian(feedData []*FeedData) (float64, error) {
	if len(feedData) == 0 {
		log.Debug().Str("Player", "Fetcher").Msg("no feed data to calculate median, probably because there are only vwap feeds")
		return 0, nil
	}

	prices := []float64{}
	for _, data := range feedData {
		prices = append(prices, data.Value)
	}

	return calculator.GetFloatMed(prices)
}

func isFXPricePair(name string) bool {
	return strings.Contains(ForeignExchangePricePairs, name)
}
