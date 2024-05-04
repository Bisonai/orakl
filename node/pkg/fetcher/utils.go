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
	"github.com/rs/zerolog/log"
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

func insertFeedData(ctx context.Context, feedData []FeedData) error {
	insertRows := make([][]any, 0, len(feedData))
	for _, data := range feedData {
		insertRows = append(insertRows, []any{data.FeedID, data.Value})
	}

	err := db.BulkInsert(ctx, "feed_data", []string{"feed_id", "value"}, insertRows)
	if err != nil {
		log.Error().Str("Player", "Fetcher").Err(err).Msg("failed to insert feed data")
	}
	return err
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
