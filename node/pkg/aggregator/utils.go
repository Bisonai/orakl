package aggregator

import (
	"context"

	"bisonai.com/miko/node/pkg/common/keys"
	"bisonai.com/miko/node/pkg/db"
	"bisonai.com/miko/node/pkg/metrics"
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

// publishSubmissionData publishes to redis submissionDataStream:<symbol> for
// external consumers (DAL). The node's own DB write does not use it.
func publishSubmissionData(ctx context.Context, data SubmissionData) error {
	return db.Publish(ctx, keys.SubmissionDataStreamKey(data.Symbol), data)
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

// syncGlobalAggregateMetrics keeps global aggregate gauges in line with the
// loaded configs (drops removed ones).
func syncGlobalAggregateMetrics(configs []Config) {
	names := make([]string, 0, len(configs))
	for _, config := range configs {
		names = append(names, config.Name)
	}
	metrics.SetGlobalAggregateConfigs(names)
}
