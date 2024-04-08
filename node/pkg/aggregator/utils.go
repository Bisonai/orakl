package aggregator

import (
	"context"
	"encoding/json"

	"bisonai.com/orakl/node/pkg/db"
)

func GetLatestLocalAggregateFromRdb(ctx context.Context, name string) (redisLocalAggregate, error) {
	key := "localAggregate:" + name
	var aggregate redisLocalAggregate
	data, err := db.Get(ctx, key)
	if err != nil {
		return aggregate, err
	}

	err = json.Unmarshal([]byte(data), &aggregate)
	if err != nil {
		return aggregate, err
	}
	return aggregate, nil
}

func GetLatestLocalAggregateFromPgs(ctx context.Context, name string) (pgsLocalAggregate, error) {
	return db.QueryRow[pgsLocalAggregate](ctx, SelectLatestLocalAggregateQuery, map[string]any{"name": name})
}

func GetLatestGlobalAggregateFromRdb(ctx context.Context, name string) (globalAggregate, error) {
	key := "globalAggregate:" + name
	var aggregate globalAggregate
	data, err := db.Get(ctx, key)
	if err != nil {
		return aggregate, err
	}

	err = json.Unmarshal([]byte(data), &aggregate)
	if err != nil {
		return aggregate, err
	}
	return aggregate, nil
}

func FilterNegative(values []int64) []int64 {
	result := []int64{}
	for _, value := range values {
		if value < 0 {
			continue
		}
		result = append(result, value)
	}
	return result
}
