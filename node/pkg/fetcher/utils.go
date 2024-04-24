package fetcher

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"time"

	"bisonai.com/orakl/node/pkg/db"
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
		return 0, errors.New("invalid input")
	}

	sqrtPriceX96Float := new(big.Float).SetInt(sqrtPriceX96)
	sqrtPriceX96Float.Quo(sqrtPriceX96Float, new(big.Float).SetFloat64(math.Pow(2, 96)))
	sqrtPriceX96Float.Mul(sqrtPriceX96Float, sqrtPriceX96Float) // square

	decimalDiff := new(big.Float).SetFloat64(math.Pow(10, float64(decimal1-decimal0)))

	datum := sqrtPriceX96Float.Quo(sqrtPriceX96Float, decimalDiff)
	if definition.Reciprocal != nil && *definition.Reciprocal {
		if datum == nil || datum.Sign() == 0 {
			return 0, errors.New("division by zero error from reciprocal division")
		}
		datum = datum.Quo(new(big.Float).SetFloat64(1), datum)
	}

	multiplier := new(big.Float).SetFloat64(math.Pow(10, DECIMALS))
	datum.Mul(datum, multiplier)

	result, _ := datum.Float64()

	return math.Round(result), nil
}

func insertFeedData(ctx context.Context, adapterId int64, feedData []FeedData) error {
	insertRows := make([][]any, 0, len(feedData))
	for _, data := range feedData {
		insertRows = append(insertRows, []any{adapterId, data.FeedName, data.Value})
	}

	err := db.BulkInsert(ctx, "feed_data", []string{"adapter_id", "name", "value"}, insertRows)
	if err != nil {
		log.Error().Str("Player", "Fetcher").Err(err).Msg("failed to insert feed data")
	}
	return err
}

func insertPgsql(ctx context.Context, name string, value float64) error {
	err := db.QueryWithoutResult(ctx, InsertLocalAggregateQuery, map[string]any{"name": name, "value": int64(value)})
	return err
}

func insertRdb(ctx context.Context, name string, value float64) error {
	key := "localAggregate:" + name
	data, err := json.Marshal(redisAggregate{Value: int64(value), Timestamp: time.Now()})
	if err != nil {
		return err
	}
	return db.Set(ctx, key, string(data), time.Duration(5*time.Minute))
}
