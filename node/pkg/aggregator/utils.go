package aggregator

import (
	"context"

	"bisonai.com/miko/node/pkg/common/keys"
	"bisonai.com/miko/node/pkg/db"
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

// newSubmissionData builds the payload shared by the DB bulk writer and the
// redis submissionDataStream; ok is false for empty aggregates, which are
// neither stored nor published.
func newSubmissionData(name string, globalAggregate GlobalAggregate, proof Proof) (SubmissionData, bool) {
	if globalAggregate.Value == 0 || globalAggregate.Timestamp.IsZero() {
		return SubmissionData{}, false
	}
	return SubmissionData{
		Symbol:          name,
		GlobalAggregate: globalAggregate,
		Proof:           proof,
	}, true
}

func PublishGlobalAggregateAndProof(ctx context.Context, name string, globalAggregate GlobalAggregate, proof Proof) error {
	data, ok := newSubmissionData(name, globalAggregate, proof)
	if !ok {
		return nil
	}
	return db.Publish(ctx, keys.SubmissionDataStreamKey(name), data)
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
