package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"github.com/rs/zerolog/log"
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

func InsertGlobalAggregate(ctx context.Context, name string, value int64, round int64) error {
	var errs []string

	err := insertRdb(ctx, name, value, round)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to insert global aggregate into rdb")
		errs = append(errs, err.Error())
	}

	err = insertPgsql(ctx, name, value, round)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to insert global aggregate into pgsql")
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "; "))
	}

	return nil
}

func insertPgsql(ctx context.Context, name string, value int64, round int64) error {
	return db.QueryWithoutResult(ctx, InsertGlobalAggregateQuery, map[string]any{"name": name, "value": value, "round": round})
}

func insertRdb(ctx context.Context, name string, value int64, round int64) error {
	key := "globalAggregate:" + name
	data, err := json.Marshal(globalAggregate{Name: name, Value: value, Round: round})
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to marshal global aggregate")
		return err
	}
	return db.Set(ctx, key, string(data), time.Duration(5*time.Minute))
}

func InsertProof(ctx context.Context, name string, round int64, proofs [][]byte) error {
	var errs []string

	err := insertProofRdb(ctx, name, round, proofs)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to insert proof into rdb")
		errs = append(errs, err.Error())
	}

	err = insertProofPgsql(ctx, name, round, proofs)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to insert proof into pgsql")
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "; "))
	}

	return nil
}

func insertProofPgsql(ctx context.Context, name string, round int64, proofs [][]byte) error {
	insertRows := make([][]any, 0, len(proofs))
	for _, proof := range proofs {
		insertRows = append(insertRows, []any{name, round, proof})
	}

	_, err := db.BulkInsert(ctx, "proofs", []string{"name", "round", "proof"}, insertRows)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to insert proofs into pgsql")
	}
	return err
}

func insertProofRdb(ctx context.Context, name string, round int64, proofs [][]byte) error {
	key := "proof:" + name + "|round:" + strconv.FormatInt(round, 10)
	data, err := json.Marshal(Proofs{Name: name, Round: round, Proofs: proofs})
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to marshal proofs")
		return err
	}
	return db.Set(ctx, key, string(data), time.Duration(5*time.Minute))
}

func GetLatestLocalAggregate(ctx context.Context, name string) (int64, time.Time, error) {
	redisAggregate, err := GetLatestLocalAggregateFromRdb(ctx, name)
	if err != nil {
		pgsqlAggregate, err := GetLatestLocalAggregateFromPgs(ctx, name)
		if err != nil {
			return 0, time.Time{}, err
		}
		return pgsqlAggregate.Value, pgsqlAggregate.Timestamp, nil
	}
	return redisAggregate.Value, redisAggregate.Timestamp, nil
}
