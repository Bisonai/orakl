package fetcher

import (
	"context"
	"strings"

	"bisonai.com/orakl/node/pkg/common/keys"
	"bisonai.com/orakl/node/pkg/db"
	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/utils/calculator"
	"bisonai.com/orakl/node/pkg/utils/reducer"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/rs/zerolog/log"
)

func FetchSingle(ctx context.Context, definition *Definition) (float64, error) {
	rawResult, err := request.Request[interface{}](request.WithEndpoint(*definition.Url), request.WithHeaders(definition.Headers))
	if err != nil {
		return 0, err
	}
	return reducer.Reduce(rawResult, definition.Reducers)
}

func getLatestFeedData(ctx context.Context, feedIds []int32) ([]FeedData, error) {
	if len(feedIds) == 0 {
		return []*FeedData{}, nil
	}
	keyList := make([]string, len(feedIds))
	for i, feedId := range feedIds {
		keyList[i] = keys.LatestFeedDataKey(feedId)
	}
	feedData, err := db.MGetObject[*FeedData](ctx, keyList)
	if err != nil {
		return nil, err
	}

	return feedData, nil
}

func setFeedDataBuffer(ctx context.Context, feedData []*FeedData) error {
	return db.LPushObject(ctx, keys.FeedDataBufferKey(), feedData)
}

func getFeedDataBuffer(ctx context.Context) ([]*FeedData, error) {
	// buffer flushed on pop all
	return db.PopAllObject[*FeedData](ctx, keys.FeedDataBufferKey())
}

func copyFeedData(ctx context.Context, feedData []*FeedData) error {
	if len(feedData) == 0 {
		return nil
	}
	insertRows := make([][]any, len(feedData))
	for i, data := range feedData {
		insertRows[i] = []any{data.FeedID, data.Value, data.Timestamp, data.Volume}
	}
	_, err := db.BulkCopy(ctx, "feed_data", []string{"feed_id", "value", "timestamp", "volume"}, insertRows)
	return err
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
		log.Debug().Str("Player", "Fetcher").Msg("no feed data to calculate median")
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
