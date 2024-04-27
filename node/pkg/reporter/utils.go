package reporter

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"strconv"
	"time"

	"bisonai.com/orakl/node/pkg/chain/helper"
	chain_utils "bisonai.com/orakl/node/pkg/chain/utils"
	"bisonai.com/orakl/node/pkg/db"

	"github.com/klaytn/klaytn/common"
	"github.com/rs/zerolog/log"
)

func GetDeviatingAggregates(lastSubmitted []GlobalAggregate, newAggregates []GlobalAggregate) []GlobalAggregate {
	submittedAggregates := make(map[int32]GlobalAggregate)
	for _, aggregate := range lastSubmitted {
		submittedAggregates[aggregate.ConfigID] = aggregate
	}

	result := make([]GlobalAggregate, 0, len(newAggregates))
	for _, newAggregate := range newAggregates {
		submittedAggregate, ok := submittedAggregates[newAggregate.ConfigID]
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

func GetLastSubmission(ctx context.Context, submissionPairs map[int32]SubmissionPair) ([]GlobalAggregate, error) {
	keys := make([]string, 0, len(submissionPairs))

	for config_id := range submissionPairs {
		keys = append(keys, "lastSubmission:"+strconv.Itoa(int(config_id)))
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
		if agg.ConfigID == 0 {
			log.Error().Str("Player", "Reporter").Int32("name", agg.ConfigID).Msg("skipping invalid aggregate")
			continue
		}
		key := "lastSubmission:" + strconv.Itoa(int(agg.ConfigID))

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

func ProofsToMap(proofs []Proof) map[int32][]byte {
	m := make(map[int32][]byte)
	for _, proof := range proofs {
		m[proof.ConfigID] = proof.Proof
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
			ConfigID: pgsqlProof.ConfigID,
			Round:    pgsqlProof.Round,
			Proof:    pgsqlProof.Proof,
		}
	}
	return proofs
}

func MakeContractArgsWithProofs(aggregates []GlobalAggregate, submissionPairs map[int32]SubmissionPair, proofMap map[int32][]byte) ([]common.Address, []*big.Int, [][]byte, []*big.Int, error) {
	addresses := make([]common.Address, len(aggregates))
	values := make([]*big.Int, len(aggregates))
	proofs := make([][]byte, len(aggregates))
	timestamps := make([]*big.Int, len(aggregates))

	for i, agg := range aggregates {
		if agg.ConfigID == 0 || agg.Value < 0 {
			log.Error().Str("Player", "Reporter").Int32("configId", agg.ConfigID).Int64("value", agg.Value).Msg("skipping invalid aggregate")
			return nil, nil, nil, nil, errors.New("invalid aggregate exists")
		}
		addresses[i] = submissionPairs[agg.ConfigID].Address
		values[i] = big.NewInt(agg.Value)
		proofs[i] = proofMap[agg.ConfigID]
		timestamps[i] = big.NewInt(agg.Timestamp.Unix())
	}

	if len(addresses) == 0 || len(values) == 0 || len(proofs) == 0 || len(timestamps) == 0 {
		return nil, nil, nil, nil, errors.New("no valid aggregates")
	}
	return addresses, values, proofs, timestamps, nil
}

func MakeContractArgsWithoutProofs(aggregates []GlobalAggregate, submissionPairs map[int32]SubmissionPair) ([]common.Address, []*big.Int, error) {
	addresses := make([]common.Address, len(aggregates))
	values := make([]*big.Int, len(aggregates))

	for i, agg := range aggregates {
		if agg.ConfigID == 0 || agg.Value < 0 {
			log.Error().Str("Player", "Reporter").Int32("configId", agg.ConfigID).Int64("value", agg.Value).Msg("skipping invalid aggregate")
			return nil, nil, errors.New("invalid aggregate exists")
		}
		addresses[i] = submissionPairs[agg.ConfigID].Address
		values[i] = big.NewInt(agg.Value)

	}

	if len(addresses) == 0 || len(values) == 0 {
		return nil, nil, errors.New("no valid aggregates")
	}
	return addresses, values, nil
}

func FilterInvalidAggregates(aggregates []GlobalAggregate, submissionPairs map[int32]SubmissionPair) []GlobalAggregate {
	validAggregates := make([]GlobalAggregate, 0, len(aggregates))
	for _, aggregate := range aggregates {
		if IsAggValid(aggregate, submissionPairs) {
			validAggregates = append(validAggregates, aggregate)
		}
	}
	return validAggregates
}

func IsAggValid(aggregate GlobalAggregate, submissionPairs map[int32]SubmissionPair) bool {
	lastSubmission := submissionPairs[aggregate.ConfigID].LastSubmission
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
		keys = append(keys, "proof:"+strconv.Itoa(int(agg.ConfigID))+"|round:"+strconv.FormatInt(agg.Round, 10))
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

func GetProofsAsMap(ctx context.Context, aggregates []GlobalAggregate) (map[int32][]byte, error) {
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
	keys := make([]string, 0, len(submissionPairs))

	for configId := range submissionPairs {
		keys = append(keys, "globalAggregate:"+strconv.Itoa(int(configId)))
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

func ReadOnchainWhitelist(ctx context.Context, chainHelper *helper.ChainHelper, contractAddress string, contractFunction string) ([]common.Address, error) {
	result, err := chainHelper.ReadContract(ctx, contractAddress, contractFunction)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to read contract")
		return nil, err
	}

	rawResultSlice, ok := result.([]interface{})
	if !ok {
		log.Error().Str("Player", "Reporter").Msg("unexpected raw result type")
		return nil, errors.New("unexpected result type")
	}

	arr, ok := rawResultSlice[0].([]common.Address)
	if !ok {
		log.Error().Str("Player", "Reporter").Msg("unexpected raw result type")
		return nil, errors.New("unexpected rawResult type")
	}
	return arr, nil
}

func CheckForNonWhitelistedSigners(signers []common.Address, whitelist []common.Address) error {
	for _, signer := range signers {
		if !isWhitelisted(signer, whitelist) {
			log.Error().Str("Player", "Reporter").Str("signer", signer.Hex()).Msg("non-whitelisted signer")
			return errors.New("non-whitelisted signer")
		}
	}
	return nil
}

func isWhitelisted(signer common.Address, whitelist []common.Address) bool {
	for _, w := range whitelist {
		if w == signer {
			return true
		}
	}
	return false
}

func OrderProof(signerMap map[common.Address][]byte, whitelist []common.Address) ([]byte, error) {
	tmpProofs := make([][]byte, 0, len(whitelist))
	for _, signer := range whitelist {
		tmpProof, ok := signerMap[signer]
		if ok {
			tmpProofs = append(tmpProofs, tmpProof)
		}
	}

	if len(tmpProofs) == 0 {
		log.Error().Str("Player", "Reporter").Msg("no valid proofs")
		return nil, errors.New("no valid proofs")
	}

	return bytes.Join(tmpProofs, nil), nil
}

func GetSignerMap(signers []common.Address, proofChunks [][]byte) map[common.Address][]byte {
	signerMap := make(map[common.Address][]byte)
	for i, signer := range signers {
		signerMap[signer] = proofChunks[i]
	}
	return signerMap
}

func GetSignerListFromProofs(hash []byte, proofChunks [][]byte) ([]common.Address, error) {
	signers := make([]common.Address, 0, len(proofChunks))
	for _, p := range proofChunks {
		signer, err := chain_utils.RecoverSigner(hash, p)
		if err != nil {
			return nil, err
		}
		signers = append(signers, signer)
	}

	return signers, nil
}

func SplitProofToChunk(proof []byte) ([][]byte, error) {
	if len(proof) == 0 {
		return nil, errors.New("empty proof")
	}

	if len(proof)%65 != 0 {
		return nil, errors.New("invalid proof length")
	}

	proofs := make([][]byte, 0, len(proof)/65)
	for i := 0; i < len(proof); i += 65 {
		proofs = append(proofs, proof[i:i+65])
	}

	return proofs, nil
}

func RemoveDuplicateProof(proof []byte) []byte {
	proofs, err := SplitProofToChunk(proof)
	if err != nil {
		return []byte{}
	}

	uniqueProofs := make(map[string][]byte)
	for _, p := range proofs {
		uniqueProofs[string(p)] = p
	}

	result := make([][]byte, 0, len(uniqueProofs)*65)
	for _, p := range uniqueProofs {
		result = append(result, p)
	}

	return bytes.Join(result, nil)
}

func UpsertProofs(ctx context.Context, aggregates []GlobalAggregate, proofMap map[int32][]byte) error {
	upsertRows := make([][]any, 0, len(aggregates))
	for _, agg := range aggregates {
		proof, ok := proofMap[agg.ConfigID]
		if !ok {
			continue
		}
		upsertRows = append(upsertRows, []any{agg.ConfigID, agg.Round, proof})
	}

	err := db.BulkUpsert(ctx, "proofs", []string{"config_id", "round", "proof"}, upsertRows, []string{"config_id", "round"}, []string{"proof"})
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to upsert proofs")
	}
	return err
}

func UpdateProofs(ctx context.Context, aggregates []GlobalAggregate, proofMap map[int32][]byte) error {
	rows := make([][]any, 0, len(aggregates))
	for _, agg := range aggregates {
		proof, ok := proofMap[agg.ConfigID]
		if !ok {
			continue
		}
		rows = append(rows, []any{proof, agg.ConfigID, agg.Round})
	}

	err := db.BulkUpdate(ctx, "proofs", []string{"proof"}, rows, []string{"config_id", "round"})
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to update proofs")
	}
	return err
}
