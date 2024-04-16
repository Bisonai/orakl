package reporter

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"strconv"
	"time"

	"bisonai.com/orakl/node/pkg/db"

	"github.com/klaytn/klaytn/common"
	"github.com/rs/zerolog/log"
)

func GetDeviatingAggregates(lastSubmitted []GlobalAggregate, newAggregates []GlobalAggregate) []GlobalAggregate {
	submittedAggregates := make(map[string]GlobalAggregate)
	for _, aggregate := range lastSubmitted {
		submittedAggregates[aggregate.Name] = aggregate
	}

	result := make([]GlobalAggregate, 0, len(newAggregates))
	for _, newAggregate := range newAggregates {
		submittedAggregate, ok := submittedAggregates[newAggregate.Name]
		if !ok || ShouldReportDeviation(submittedAggregate.Value, newAggregate.Value) {
			result = append(result, newAggregate)
		}
	}
	return result
}

func ShouldReportDeviation(oldValue int64, newValue int64) bool {
	denominator := math.Pow10(DECIMALS)
	oldValueInFLoat := float64(oldValue) / denominator
	newValueInFLoat := float64(newValue) / denominator

	if oldValue != 0 && newValue != 0 {
		deviationRange := oldValueInFLoat * DEVIATION_THRESHOLD
		minimum := oldValueInFLoat - deviationRange
		maximum := oldValueInFLoat + deviationRange
		return newValueInFLoat < minimum || newValueInFLoat > maximum
	} else if oldValue == 0 && newValue != 0 {
		return newValueInFLoat > DEVIATION_ABSOLUTE_THRESHOLD
	} else {
		return false
	}
}

func GetLastSubmission(ctx context.Context, submissionPairs map[string]SubmissionPair) ([]GlobalAggregate, error) {
	keys := make([]string, 0, len(submissionPairs))

	for name := range submissionPairs {
		keys = append(keys, "lastSubmission:"+name)
	}

	result, err := db.MGet(ctx, keys)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to get last submission")
		return nil, err
	}

	aggregates := make([]GlobalAggregate, 0, len(result))
	for i, agg := range result {
		if agg == nil {
			log.Error().Str("Player", "Reporter").Str("key", keys[i]).Msg("missing aggregate")
			continue
		}
		var aggregate GlobalAggregate
		err = json.Unmarshal([]byte(agg.(string)), &aggregate)
		if err != nil {
			log.Error().Str("Player", "Reporter").Err(err).Str("key", keys[i]).Msg("failed to unmarshal aggregate")
			continue
		}
		aggregates = append(aggregates, aggregate)
	}

	return aggregates, nil
}

func StoreLastSubmission(ctx context.Context, aggregates []GlobalAggregate) error {
	vals := make(map[string]string)

	for _, agg := range aggregates {
		if agg.Name == "" {
			log.Error().Str("Player", "Reporter").Str("name", agg.Name).Msg("skipping invalid aggregate")
			continue
		}
		key := "lastSubmission:" + agg.Name

		tmpValue, err := json.Marshal(agg)
		if err != nil {
			log.Error().Str("Player", "Reporter").Err(err).Msg("failed to marshal aggregate")
			continue
		}
		value := string(tmpValue)
		vals[key] = value
	}

	if len(vals) == 0 {
		return errors.New("no valid aggregates")
	}
	return db.MSet(ctx, vals)
}

func ProofsToMap(proofs []Proof) map[string][]byte {
	m := make(map[string][]byte)
	for _, proof := range proofs {
		//m[name-round] = proof
		m[proof.Name+"-"+strconv.FormatInt(proof.Round, 10)] = proof.Proof
	}
	return m
}

func CalculateJitter(baseTimeout time.Duration) time.Duration {
	n, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to generate jitter for retry timeout")
		return baseTimeout
	}
	jitter := time.Duration(n.Int64()) * time.Millisecond
	return baseTimeout + jitter
}

func ConvertPgsqlProofsToProofs(pgsqlProofs []PgsqlProof) []Proof {
	proofs := make([]Proof, len(pgsqlProofs))
	for i, pgsqlProof := range pgsqlProofs {
		proofs[i] = Proof{
			Name:  pgsqlProof.Name,
			Round: pgsqlProof.Round,
			Proof: pgsqlProof.Proof,
		}
	}
	return proofs
}

func MakeContractArgsWithProofs(aggregates []GlobalAggregate, submissionPairs map[string]SubmissionPair, proofMap map[string][]byte) ([]common.Address, []*big.Int, [][]byte, []*big.Int, error) {
	addresses := make([]common.Address, len(aggregates))
	values := make([]*big.Int, len(aggregates))
	proofs := make([][]byte, len(aggregates))
	timestamps := make([]*big.Int, len(aggregates))

	for i, agg := range aggregates {
		if agg.Name == "" || agg.Value < 0 {
			log.Error().Str("Player", "Reporter").Str("name", agg.Name).Int64("value", agg.Value).Msg("skipping invalid aggregate")
			return nil, nil, nil, nil, errors.New("invalid aggregate exists")
		}
		addresses[i] = submissionPairs[agg.Name].Address
		values[i] = big.NewInt(agg.Value)
		proofs[i] = proofMap[agg.Name+"-"+strconv.FormatInt(agg.Round, 10)]
		timestamps[i] = big.NewInt(agg.Timestamp.Unix())
	}

	if len(addresses) == 0 || len(values) == 0 || len(proofs) == 0 || len(timestamps) == 0 {
		return nil, nil, nil, nil, errors.New("no valid aggregates")
	}
	return addresses, values, proofs, timestamps, nil
}

func MakeContractArgsWithoutProofs(aggregates []GlobalAggregate, submissionPairs map[string]SubmissionPair) ([]common.Address, []*big.Int, error) {
	addresses := make([]common.Address, len(aggregates))
	values := make([]*big.Int, len(aggregates))

	for i, agg := range aggregates {
		if agg.Name == "" || agg.Value < 0 {
			log.Error().Str("Player", "Reporter").Str("name", agg.Name).Int64("value", agg.Value).Msg("skipping invalid aggregate")
			return nil, nil, errors.New("invalid aggregate exists")
		}
		addresses[i] = submissionPairs[agg.Name].Address
		values[i] = big.NewInt(agg.Value)

	}

	if len(addresses) == 0 || len(values) == 0 {
		return nil, nil, errors.New("no valid aggregates")
	}
	return addresses, values, nil
}

func FilterInvalidAggregates(aggregates []GlobalAggregate, submissionPairs map[string]SubmissionPair) []GlobalAggregate {
	validAggregates := make([]GlobalAggregate, 0, len(aggregates))
	for _, aggregate := range aggregates {
		if IsAggValid(aggregate, submissionPairs) {
			validAggregates = append(validAggregates, aggregate)
		}
	}
	return validAggregates
}

func IsAggValid(aggregate GlobalAggregate, submissionPairs map[string]SubmissionPair) bool {
	lastSubmission := submissionPairs[aggregate.Name].LastSubmission
	if lastSubmission == 0 {
		return true
	}
	return aggregate.Round > lastSubmission
}

func GetProofs(ctx context.Context, aggregates []GlobalAggregate) ([]Proof, error) {
	result, err := GetProofsRdb(ctx, aggregates)
	if err != nil {
		log.Warn().Str("Player", "Reporter").Err(err).Msg("getProofsRdb failed, trying to get from pgsql")
		return GetProofsPgsql(ctx, aggregates)
	}
	return result, nil
}

func GetProofsRdb(ctx context.Context, aggregates []GlobalAggregate) ([]Proof, error) {
	keys := make([]string, 0, len(aggregates))
	for _, agg := range aggregates {
		keys = append(keys, "proof:"+agg.Name+"|round:"+strconv.FormatInt(agg.Round, 10))
	}

	result, err := db.MGet(ctx, keys)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to get proofs")
		return nil, err
	}

	proofs := make([]Proof, 0, len(result))
	for i, proof := range result {
		if proof == nil {
			log.Error().Str("Player", "Reporter").Str("key", keys[i]).Msg("missing proof")
			continue
		}
		var p Proof
		err = json.Unmarshal([]byte(proof.(string)), &p)
		if err != nil {
			log.Error().Str("Player", "Reporter").Err(err).Str("key", keys[i]).Msg("failed to unmarshal proof")
			continue
		}
		proofs = append(proofs, p)

	}
	return proofs, nil
}

func GetProofsPgsql(ctx context.Context, aggregates []GlobalAggregate) ([]Proof, error) {
	q := makeGetProofsQuery(aggregates)
	rawResult, err := db.QueryRows[PgsqlProof](ctx, q, nil)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to get proofs")
		return nil, err
	}
	return ConvertPgsqlProofsToProofs(rawResult), nil
}

func GetProofsAsMap(ctx context.Context, aggregates []GlobalAggregate) (map[string][]byte, error) {
	proofs, err := GetProofs(ctx, aggregates)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("submit without proofs")
		return nil, err
	}

	if len(proofs) < len(aggregates) {
		log.Error().Str("Player", "Reporter").Msg("proofs not found for all aggregates")
		return nil, errors.New("proofs not found for all aggregates")
	}
	return ProofsToMap(proofs), nil
}

func GetLatestGlobalAggregates(ctx context.Context, submissionPairs map[string]SubmissionPair) ([]GlobalAggregate, error) {
	result, err := GetLatestGlobalAggregatesRdb(ctx, submissionPairs)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("getLatestGlobalAggregatesRdb failed, trying to get from pgsql")
		return GetLatestGlobalAggregatesPgsql(ctx, submissionPairs)
	}
	return result, nil
}

func GetLatestGlobalAggregatesPgsql(ctx context.Context, submissionPairs map[string]SubmissionPair) ([]GlobalAggregate, error) {
	names := make([]string, 0, len(submissionPairs))
	for name := range submissionPairs {
		names = append(names, name)
	}

	q := makeGetLatestGlobalAggregatesQuery(names)
	return db.QueryRows[GlobalAggregate](ctx, q, nil)
}

func GetLatestGlobalAggregatesRdb(ctx context.Context, submissionPairs map[string]SubmissionPair) ([]GlobalAggregate, error) {
	keys := make([]string, 0, len(submissionPairs))

	for name := range submissionPairs {
		keys = append(keys, "globalAggregate:"+name)
	}

	result, err := db.MGet(ctx, keys)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to get latest global aggregates")
		return nil, err
	}

	aggregates := make([]GlobalAggregate, 0, len(result))
	for i, agg := range result {
		if agg == nil {
			log.Warn().Str("Player", "Reporter").Str("key", keys[i]).Msg("no latest aggregate")
			continue
		}
		var aggregate GlobalAggregate
		err = json.Unmarshal([]byte(agg.(string)), &aggregate)
		if err != nil {
			log.Error().Str("Player", "Reporter").Err(err).Str("key", keys[i]).Msg("failed to unmarshal aggregate")
			continue
		}
		aggregates = append(aggregates, aggregate)
	}
	return aggregates, nil
}

func ValidateAggregateTimestampValues(aggregates []GlobalAggregate) bool {
	for _, agg := range aggregates {
		if agg.Timestamp.IsZero() || agg.Timestamp.After(time.Now()) {
			return false
		}
	}
	return true
}
