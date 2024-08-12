package aggregator

import (
	"context"

	"bisonai.com/orakl/node/pkg/common/keys"
	"bisonai.com/orakl/node/pkg/db"
)

func FilterNegative(values []int64) []int64 {
	result := make([]int64, 0, len(values))
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

	return db.Publish(ctx, keys.SubmissionDataStreamKey(globalAggregate.ConfigID), data)
}

func getLatestRoundId(ctx context.Context, configId int32) (int32, error) {
	result, err := db.QueryRow[GlobalAggregate](ctx, SelectLatestGlobalAggregateQuery, map[string]any{"config_id": configId})
	if err != nil {
		return 0, err
	}
	return result.Round, nil
}

// used for testing
func getProofFromPgs(ctx context.Context, configId int32, round int32) (Proof, error) {
	return db.QueryRow[Proof](ctx, "SELECT config_id, round, proof FROM proofs WHERE config_id = @config_id AND round = @round", map[string]any{"config_id": configId, "round": round})
}

// used for testing
func getLatestGlobalAggregateFromPgs(ctx context.Context, configId int32) (GlobalAggregate, error) {
	return db.QueryRow[GlobalAggregate](ctx, SelectLatestGlobalAggregateQuery, map[string]any{"config_id": configId})
}
