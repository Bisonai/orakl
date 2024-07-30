package aggregator

import (
	"context"
	"time"

	"bisonai.com/orakl/node/pkg/common/keys"
	"bisonai.com/orakl/node/pkg/db"
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

func PublishGlobalAggregateAndProof(ctx context.Context, globalAggregate GlobalAggregate, proof Proof) error {
	if globalAggregate.Value == 0 || globalAggregate.Timestamp.IsZero() {
		return nil
	}
	data := SubmissionData{
		GlobalAggregate: globalAggregate,
		Proof:           proof,
	}

	diff := time.Since(globalAggregate.Timestamp)
	if diff > 1*time.Second {
		log.Info().Dur("duration", diff).Int32("config_id", globalAggregate.ConfigID).Int64("value", globalAggregate.Value).Msg("published global aggregate")

	}
	return db.Publish(ctx, keys.SubmissionDataStreamKey(globalAggregate.ConfigID), data)
}

func GetLatestLocalAggregate(ctx context.Context, configId int32) (int64, time.Time, error) {
	redisAggregate, err := GetLatestLocalAggregateFromRdb(ctx, configId)
	diff := time.Since(redisAggregate.Timestamp).Milliseconds()
	if diff > 400 {
		log.Warn().Msgf("redisAggregate is %d ms old", diff)
	}
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
func getLatestGlobalAggregateFromRdb(ctx context.Context, configId int32) (GlobalAggregate, error) {
	return db.GetObject[GlobalAggregate](ctx, keys.GlobalAggregateKey(configId))
}

// used for testing
func getProofFromPgs(ctx context.Context, configId int32, round int32) (Proof, error) {
	return db.QueryRow[Proof](ctx, "SELECT config_id, round, proof FROM proofs WHERE config_id = @config_id AND round = @round", map[string]any{"config_id": configId, "round": round})
}

// used for testing
func getLatestGlobalAggregateFromPgs(ctx context.Context, configId int32) (GlobalAggregate, error) {
	return db.QueryRow[GlobalAggregate](ctx, SelectLatestGlobalAggregateQuery, map[string]any{"config_id": configId})
}
