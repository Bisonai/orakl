package aggregator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"github.com/rs/zerolog/log"
)

func GetLatestLocalAggregateFromRdb(ctx context.Context, name string) (LocalAggregate, error) {
	key := "localAggregate:" + name
	var aggregate LocalAggregate
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

func GetLatestLocalAggregateFromPgs(ctx context.Context, name string) (PgsLocalAggregate, error) {
	return db.QueryRow[PgsLocalAggregate](ctx, SelectLatestLocalAggregateQuery, map[string]any{"name": name})
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

func InsertGlobalAggregate(ctx context.Context, name string, value int64, round int64, timestamp time.Time) error {
	var errs []string

	err := insertRdb(ctx, name, value, round, timestamp)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to insert global aggregate into rdb")
		errs = append(errs, err.Error())
	}

	err = insertPgsql(ctx, name, value, round, timestamp)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to insert global aggregate into pgsql")
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "; "))
	}

	return nil
}

func insertPgsql(ctx context.Context, name string, value int64, round int64, timestamp time.Time) error {
	return db.QueryWithoutResult(ctx, InsertGlobalAggregateQuery, map[string]any{"name": name, "value": value, "round": round, "timestamp": timestamp})
}

func insertRdb(ctx context.Context, name string, value int64, round int64, timestamp time.Time) error {
	key := "globalAggregate:" + name
	data, err := json.Marshal(GlobalAggregate{Name: name, Value: value, Round: round, Timestamp: timestamp})
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
	concatProof := bytes.Join(proofs, nil)
	err := db.QueryWithoutResult(ctx, InsertProofQuery, map[string]any{"name": name, "round": round, "proof": concatProof})
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to insert proofs into pgsql")
	}

	return err
}

func insertProofRdb(ctx context.Context, name string, round int64, proofs [][]byte) error {
	concatProof := bytes.Join(proofs, nil)
	key := "proof:" + name + "|round:" + strconv.FormatInt(round, 10)
	data, err := json.Marshal(Proof{Name: name, Round: round, Proof: concatProof})
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

func getLatestRoundId(ctx context.Context, name string) (int64, error) {
	result, err := db.QueryRow[GlobalAggregate](ctx, SelectLatestGlobalAggregateQuery, map[string]any{"name": name})
	if err != nil {
		return 0, err
	}
	return result.Round, nil
}

// used for testing
func getProofFromRdb(ctx context.Context, name string, round int64) (Proof, error) {
	key := "proof:" + name + "|round:" + strconv.FormatInt(round, 10)
	var proofs Proof
	data, err := db.Get(ctx, key)
	if err != nil {
		return proofs, err
	}

	err = json.Unmarshal([]byte(data), &proofs)
	if err != nil {
		return proofs, err
	}
	return proofs, nil
}

// used for testing
func getProofFromPgsql(ctx context.Context, name string, round int64) (Proof, error) {
	rawProof, err := db.QueryRow[PgsqlProof](ctx, "SELECT * FROM proofs WHERE name = @name AND round = @round", map[string]any{"name": name, "round": round})
	if err != nil {
		return Proof{}, err
	}

	proofs := Proof{Name: name, Round: round, Proof: rawProof.Proof}
	return proofs, nil
}

// used for testing
func getLatestGlobalAggregateFromRdb(ctx context.Context, name string) (GlobalAggregate, error) {
	key := "globalAggregate:" + name
	var aggregate GlobalAggregate
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
