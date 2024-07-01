package aggregator

import (
	"bytes"
	"context"
	"time"

	"bisonai.com/orakl/node/pkg/common/keys"
	"bisonai.com/orakl/node/pkg/db"
	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"github.com/rs/zerolog/log"
)

func GetLatestLocalAggregateFromRdb(ctx context.Context, configId int32) (LocalAggregate, error) {
	return db.GetObject[LocalAggregate](ctx, keys.LocalAggregateKey(configId))
}

func GetLatestLocalAggregateFromPgs(ctx context.Context, configId int32) (LocalAggregate, error) {
	return db.QueryRow[LocalAggregate](ctx, SelectLatestLocalAggregateQuery, map[string]any{"config_id": configId})
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

func InsertGlobalAggregate(ctx context.Context, configId int32, value int64, round int32, timestamp time.Time) error {
	var errs []string

	if value == 0 || timestamp.IsZero() {
		return errorSentinel.ErrAggregatorInvalidGlobalAggInsertion
	}

	err := insertRdb(ctx, configId, value, round, timestamp)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to insert global aggregate into rdb")
		errs = append(errs, err.Error())
	}

	err = insertPgsql(ctx, configId, value, round, timestamp)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to insert global aggregate into pgsql")
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return errorSentinel.ErrAggregatorGlobalAggregateInsertion
	}

	return nil
}

func insertPgsql(ctx context.Context, configId int32, value int64, round int32, timestamp time.Time) error {
	return db.QueryWithoutResult(ctx, InsertGlobalAggregateQuery, map[string]any{"config_id": configId, "value": value, "round": round, "timestamp": timestamp})
}

func insertRdb(ctx context.Context, configId int32, value int64, round int32, timestamp time.Time) error {
	data := GlobalAggregate{ConfigID: configId, Value: value, Round: round, Timestamp: timestamp}
	return db.SetObject(ctx, keys.GlobalAggregateKey(configId), data, time.Duration(5*time.Minute))
}

func InsertProof(ctx context.Context, configId int32, round int32, proofs [][]byte) error {
	var errs []string

	err := insertProofRdb(ctx, configId, round, proofs)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to insert proof into rdb")
		errs = append(errs, err.Error())
	}

	err = insertProofPgsql(ctx, configId, round, proofs)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to insert proof into pgsql")
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return errorSentinel.ErrAggregatorProofInsertion
	}

	return nil
}

func insertProofPgsql(ctx context.Context, configId int32, round int32, proofs [][]byte) error {
	concatProof := bytes.Join(proofs, nil)
	err := db.QueryWithoutResult(ctx, InsertProofQuery, map[string]any{"config_id": configId, "round": round, "proof": concatProof})
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to insert proofs into pgsql")
	}

	return err
}

func insertProofRdb(ctx context.Context, configId int32, round int32, proofs [][]byte) error {
	concatProof := bytes.Join(proofs, nil)
	key := keys.ProofKey(configId, round)
	data := Proof{ConfigID: configId, Round: round, Proof: concatProof}
	return db.SetObject(ctx, key, data, time.Duration(5*time.Minute))
}

func GetLatestLocalAggregate(ctx context.Context, configId int32) (int64, time.Time, error) {
	redisAggregate, err := GetLatestLocalAggregateFromRdb(ctx, configId)
	if err != nil {
		pgsqlAggregate, err := GetLatestLocalAggregateFromPgs(ctx, configId)
		if err != nil {
			return 0, time.Time{}, err
		}
		return pgsqlAggregate.Value, pgsqlAggregate.Timestamp, nil
	}
	return redisAggregate.Value, redisAggregate.Timestamp, nil
}

func getLatestRoundId(ctx context.Context, configId int32) (int32, error) {
	result, err := db.QueryRow[GlobalAggregate](ctx, SelectLatestGlobalAggregateQuery, map[string]any{"config_id": configId})
	if err != nil {
		return 0, err
	}
	return result.Round, nil
}

// used for testing
func getProofFromRdb(ctx context.Context, configId int32, round int32) (Proof, error) {
	return db.GetObject[Proof](ctx, keys.ProofKey(configId, round))
}

// used for testing
func getProofFromPgsql(ctx context.Context, configId int32, round int32) (Proof, error) {
	rawProof, err := db.QueryRow[Proof](ctx, "SELECT * FROM proofs WHERE config_id = @config_id AND round = @round", map[string]any{"config_id": configId, "round": round})
	if err != nil {
		return Proof{}, err
	}

	proofs := Proof{ConfigID: configId, Round: round, Proof: rawProof.Proof}
	return proofs, nil
}

// used for testing
func getLatestGlobalAggregateFromRdb(ctx context.Context, configId int32) (GlobalAggregate, error) {
	return db.GetObject[GlobalAggregate](ctx, keys.GlobalAggregateKey(configId))
}
