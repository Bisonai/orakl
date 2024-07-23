package reporter

import (
	"context"
	"math"
	"math/big"
	"time"

	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/common/keys"
	"bisonai.com/orakl/node/pkg/db"
	errorSentinel "bisonai.com/orakl/node/pkg/error"

	"github.com/klaytn/klaytn/common"
	"github.com/klaytn/klaytn/crypto"
	"github.com/rs/zerolog/log"
)

func GetDeviatingAggregates(lastSubmitted []GlobalAggregate, newAggregates []GlobalAggregate, threshold float64) []GlobalAggregate {
	submittedAggregates := make(map[int32]GlobalAggregate)
	for _, aggregate := range lastSubmitted {
		submittedAggregates[aggregate.ConfigID] = aggregate
	}

	result := make([]GlobalAggregate, 0, len(newAggregates))
	for _, newAggregate := range newAggregates {
		submittedAggregate, ok := submittedAggregates[newAggregate.ConfigID]
		if !ok || ShouldReportDeviation(submittedAggregate.Value, newAggregate.Value, threshold) {
			result = append(result, newAggregate)
		}
	}
	return result
}

func ShouldReportDeviation(oldValue int64, newValue int64, threshold float64) bool {
	denominator := math.Pow10(DECIMALS)
	oldValueInFLoat := float64(oldValue) / denominator
	newValueInFLoat := float64(newValue) / denominator

	if oldValue != 0 && newValue != 0 {
		deviationRange := oldValueInFLoat * threshold
		minimum := oldValueInFLoat - deviationRange
		maximum := oldValueInFLoat + deviationRange
		return newValueInFLoat < minimum || newValueInFLoat > maximum
	} else if oldValue == 0 && newValue != 0 {
		return newValueInFLoat > DEVIATION_ABSOLUTE_THRESHOLD
	} else {
		return false
	}
}

func GetLastSubmission(ctx context.Context, submissionPairs map[int32]SubmissionPair) ([]GlobalAggregate, error) {
	keyList := make([]string, 0, len(submissionPairs))

	for configID := range submissionPairs {
		keyList = append(keyList, keys.LastSubmissionKey(configID))
	}

	return db.MGetObject[GlobalAggregate](ctx, keyList)
}

func MakeContractArgsWithProofs(aggregates []GlobalAggregate, submissionPairs map[int32]SubmissionPair) ([][32]byte, []*big.Int, []*big.Int, error) {
	if len(aggregates) == 0 {
		return nil, nil, nil, errorSentinel.ErrReporterEmptyAggregatesParam
	}

	if len(submissionPairs) == 0 {
		return nil, nil, nil, errorSentinel.ErrReporterEmptySubmissionPairsParam
	}

	feedHash := make([][32]byte, len(aggregates))
	values := make([]*big.Int, len(aggregates))
	timestamps := make([]*big.Int, len(aggregates))

	for i, agg := range aggregates {
		if agg.ConfigID == 0 || agg.Value < 0 {
			log.Error().Str("Player", "Reporter").Int32("configId", agg.ConfigID).Int64("value", agg.Value).Msg("skipping invalid aggregate")
			return nil, nil, nil, errorSentinel.ErrReporterInvalidAggregateFound
		}

		name := submissionPairs[agg.ConfigID].Name
		copy(feedHash[i][:], crypto.Keccak256([]byte(name)))
		values[i] = big.NewInt(agg.Value)
		timestamps[i] = big.NewInt(agg.Timestamp.Unix())
	}

	if len(feedHash) == 0 || len(values) == 0 || len(timestamps) == 0 {
		return nil, nil, nil, errorSentinel.ErrReporterEmptyValidAggregates
	}
	return feedHash, values, timestamps, nil
}

func GetLatestGlobalAggregates(ctx context.Context, submissionPairs map[int32]SubmissionPair) ([]GlobalAggregate, error) {
	result, err := GetLatestGlobalAggregatesRdb(ctx, submissionPairs)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("getLatestGlobalAggregatesRdb failed, trying to get from pgsql")
		return GetLatestGlobalAggregatesPgsql(ctx, submissionPairs)
	}
	return result, nil
}

func GetLatestGlobalAggregatesPgsql(ctx context.Context, submissionPairs map[int32]SubmissionPair) ([]GlobalAggregate, error) {
	configIds := make([]int32, 0, len(submissionPairs))
	for configId := range submissionPairs {
		configIds = append(configIds, configId)
	}

	q := makeGetLatestGlobalAggregatesQuery(configIds)
	return db.QueryRows[GlobalAggregate](ctx, q, nil)
}

func GetLatestGlobalAggregatesRdb(ctx context.Context, submissionPairs map[int32]SubmissionPair) ([]GlobalAggregate, error) {
	keyList := make([]string, 0, len(submissionPairs))

	for configId := range submissionPairs {
		keyList = append(keyList, keys.GlobalAggregateKey(configId))
	}

	return db.MGetObject[GlobalAggregate](ctx, keyList)
}

func ReadOnchainWhitelist(ctx context.Context, chainHelper *helper.ChainHelper, contractAddress string, contractFunction string) ([]common.Address, error) {
	result, err := chainHelper.ReadContract(ctx, contractAddress, contractFunction)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to read contract")
		return nil, err
	}

	rawResultSlice, ok := result.([]interface{})
	if !ok {
		log.Error().Str("Player", "Reporter").Msg("unexpected raw result type")
		return nil, errorSentinel.ErrReporterResultCastToInterfaceFail
	}

	arr, ok := rawResultSlice[0].([]common.Address)
	if !ok {
		log.Error().Str("Player", "Reporter").Msg("unexpected raw result type")
		return nil, errorSentinel.ErrReporterResultCastToAddressFail
	}
	return arr, nil
}

func GetDeviationThreshold(submissionInterval time.Duration) float64 {
	if submissionInterval <= 15*time.Second {
		return MIN_DEVIATION_THRESHOLD
	} else if submissionInterval >= 60*time.Minute {
		return MAX_DEVIATION_THRESHOLD
	} else {
		submissionIntervalSec := submissionInterval.Seconds()
		return MIN_DEVIATION_THRESHOLD - ((submissionIntervalSec-MIN_INTERVAL)/(MAX_INTERVAL-MIN_INTERVAL))*(MIN_DEVIATION_THRESHOLD-MAX_DEVIATION_THRESHOLD)
	}
}
