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

// newSubmissionData builds the payload for the redis submissionDataStream; ok
// is false for empty aggregates, which are not published.
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
// external consumers (DAL).
func publishSubmissionData(ctx context.Context, data SubmissionData) error {
	return db.Publish(ctx, keys.SubmissionDataStreamKey(data.Symbol), data)
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
