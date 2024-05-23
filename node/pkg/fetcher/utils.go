package fetcher

import (
	"context"
	"math"
	"math/big"
	"strconv"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/utils/reducer"
	"bisonai.com/orakl/node/pkg/utils/request"
)

func FetchSingle(ctx context.Context, definition *Definition) (float64, error) {
	rawResult, err := request.GetRequest[interface{}](*definition.Url, nil, definition.Headers)
	if err != nil {
		return 0, err
	}
	return reducer.Reduce(rawResult, definition.Reducers)
}

func getTokenPrice(sqrtPriceX96 *big.Int, definition *Definition) (float64, error) {
	decimal0 := *definition.Token0Decimals
	decimal1 := *definition.Token1Decimals
	if sqrtPriceX96 == nil || decimal0 == 0 || decimal1 == 0 {
		return 0, errorSentinel.ErrFetcherInvalidInput
	}

	sqrtPriceX96Float := new(big.Float).SetInt(sqrtPriceX96)
	sqrtPriceX96Float.Quo(sqrtPriceX96Float, new(big.Float).SetFloat64(math.Pow(2, 96)))
	sqrtPriceX96Float.Mul(sqrtPriceX96Float, sqrtPriceX96Float) // square

	decimalDiff := new(big.Float).SetFloat64(math.Pow(10, float64(decimal1-decimal0)))

	datum := sqrtPriceX96Float.Quo(sqrtPriceX96Float, decimalDiff)
	if definition.Reciprocal != nil && *definition.Reciprocal {
		if datum == nil || datum.Sign() == 0 {
			return 0, errorSentinel.ErrFetcherDivisionByZero
		}
		datum = datum.Quo(new(big.Float).SetFloat64(1), datum)
	}

	multiplier := new(big.Float).SetFloat64(math.Pow(10, DECIMALS))
	datum.Mul(datum, multiplier)

	result, _ := datum.Float64()

	return math.Round(result), nil
}

func setLatestFeedData(ctx context.Context, feedData []FeedData) error {
	latestData := make(map[string]any)
	for _, data := range feedData {
		latestData["latestFeedData:"+strconv.Itoa(int(data.FeedID))] = data
	}
	return db.MSetObject(ctx, latestData)
}

func getLatestFeedData(ctx context.Context, feedIds []int32) ([]FeedData, error) {
	keys := make([]string, len(feedIds))
	for i, feedId := range feedIds {
		keys[i] = "latestFeedData:" + strconv.Itoa(int(feedId))
	}
	feedData, err := db.MGetObject[FeedData](ctx, keys)
	if err != nil {
		return nil, err
	}

	return feedData, nil
}

func setFeedDataBuffer(ctx context.Context, feedData []FeedData) error {
	values := make([]interface{}, len(feedData))
	for i, data := range feedData {
		values[i] = data
	}
	return db.LPushObject(ctx, "feedDataBuffer", values)
}

func getFeedDataBuffer(ctx context.Context) ([]FeedData, error) {
	// beware, buffer will be flushed
	return db.PopAllObject[FeedData](ctx, "feedDataBuffer")
}

func insertLocalAggregatePgsql(ctx context.Context, configId int32, value float64) error {
	err := db.QueryWithoutResult(ctx, InsertLocalAggregateQuery, map[string]any{"config_id": configId, "value": int64(value)})
	return err
}

func insertLocalAggregateRdb(ctx context.Context, configId int32, value float64) error {
	key := "localAggregate:" + strconv.Itoa(int(configId))
	data := RedisAggregate{ConfigId: configId, Value: int64(value), Timestamp: time.Now()}
	return db.SetObject(ctx, key, data, time.Duration(5*time.Minute))
}

func copyFeedData(ctx context.Context, feedData []FeedData) error {
	insertRows := make([][]any, len(feedData))
	for i, data := range feedData {
		insertRows[i] = []any{data.FeedID, data.Value, data.Timestamp}
	}
	_, err := db.BulkCopy(ctx, "feed_data", []string{"feed_id", "value", "timestamp"}, insertRows)
	return err
}
