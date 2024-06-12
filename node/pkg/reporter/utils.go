package reporter

import (
	"bytes"
	"context"
	"math"
	"math/big"

	"bisonai.com/orakl/node/pkg/chain/helper"
	chainUtils "bisonai.com/orakl/node/pkg/chain/utils"
	"bisonai.com/orakl/node/pkg/common/keys"
	"bisonai.com/orakl/node/pkg/db"
	errorSentinel "bisonai.com/orakl/node/pkg/error"

	"github.com/klaytn/klaytn/common"
	"github.com/klaytn/klaytn/crypto"
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
	keyList := make([]string, 0, len(submissionPairs))

	for configID := range submissionPairs {
		keyList = append(keyList, keys.LastSubmissionKey(configID))
	}

	return db.MGetObject[GlobalAggregate](ctx, keyList)
}

func StoreLastSubmission(ctx context.Context, aggregates []GlobalAggregate) error {
	vals := make(map[string]any)

	for _, agg := range aggregates {
		if agg.ConfigID == 0 {
			log.Error().Str("Player", "Reporter").Int32("ConfigID", agg.ConfigID).Msg("skipping invalid aggregate")
			continue
		}
		vals[keys.LastSubmissionKey(agg.ConfigID)] = agg
	}

	if len(vals) == 0 {
		return errorSentinel.ErrReporterEmptyValidAggregates
	}
	return db.MSetObject(ctx, vals)
}

func ProofsToMap(proofs []Proof) map[int32][]byte {
	m := make(map[int32][]byte)
	for _, proof := range proofs {
		m[proof.ConfigID] = proof.Proof
	}
	return m
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

func MakeContractArgsWithProofs(aggregates []GlobalAggregate, submissionPairs map[int32]SubmissionPair, proofMap map[int32][]byte) ([][32]byte, []*big.Int, []*big.Int, [][]byte, error) {
	if len(aggregates) == 0 {
		return nil, nil, nil, nil, errorSentinel.ErrReporterEmptyAggregatesParam
	}

	if len(submissionPairs) == 0 {
		return nil, nil, nil, nil, errorSentinel.ErrReporterEmptySubmissionPairsParam
	}

	if len(proofMap) == 0 {
		return nil, nil, nil, nil, errorSentinel.ErrReporterEmptyProofParam
	}

	feedHash := make([][32]byte, len(aggregates))
	values := make([]*big.Int, len(aggregates))
	timestamps := make([]*big.Int, len(aggregates))
	proofs := make([][]byte, len(aggregates))

	for i, agg := range aggregates {
		if agg.ConfigID == 0 || agg.Value < 0 {
			log.Error().Str("Player", "Reporter").Int32("configId", agg.ConfigID).Int64("value", agg.Value).Msg("skipping invalid aggregate")
			return nil, nil, nil, nil, errorSentinel.ErrReporterInvalidAggregateFound
		}

		name := submissionPairs[agg.ConfigID].Name
		copy(feedHash[i][:], crypto.Keccak256([]byte(name)))
		values[i] = big.NewInt(agg.Value)
		timestamps[i] = big.NewInt(agg.Timestamp.Unix())
		proofs[i] = proofMap[agg.ConfigID]
	}

	if len(feedHash) == 0 || len(values) == 0 || len(proofs) == 0 || len(timestamps) == 0 {
		return nil, nil, nil, nil, errorSentinel.ErrReporterEmptyValidAggregates
	}
	return feedHash, values, timestamps, proofs, nil
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
	return lastSubmission == 0 || aggregate.Round > lastSubmission
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
	keyList := []string{}
	for _, agg := range aggregates {
		keyList = append(keyList, keys.ProofKey(agg.ConfigID, agg.Round))
	}
	return db.MGetObject[Proof](ctx, keyList)
}

func GetProofsPgsql(ctx context.Context, aggregates []GlobalAggregate) ([]Proof, error) {
	q := makeGetProofsQuery(aggregates)
	rawResult, err := db.QueryRows[PgsqlProof](ctx, q, nil)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to get proofs from pgsql")
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
		return nil, errorSentinel.ErrReporterMissingProof
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
	keyList := make([]string, 0, len(submissionPairs))

	for configId := range submissionPairs {
		keyList = append(keyList, keys.GlobalAggregateKey(configId))
	}

	return db.MGetObject[GlobalAggregate](ctx, keyList)
}

func ValidateAggregateTimestampValues(aggregates []GlobalAggregate) bool {
	for _, agg := range aggregates {
		if agg.Timestamp.IsZero() {
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
		return nil, errorSentinel.ErrReporterResultCastToInterfaceFail
	}

	arr, ok := rawResultSlice[0].([]common.Address)
	if !ok {
		log.Error().Str("Player", "Reporter").Msg("unexpected raw result type")
		return nil, errorSentinel.ErrReporterResultCastToAddressFail
	}
	return arr, nil
}

func CheckForNonWhitelistedSigners(signers []common.Address, whitelist []common.Address) error {
	for _, signer := range signers {
		if !isWhitelisted(signer, whitelist) {
			log.Error().Str("Player", "Reporter").Str("signer", signer.Hex()).Msg("non-whitelisted signer")
			return errorSentinel.ErrReporterSignerNotWhitelisted
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
		return nil, errorSentinel.ErrReporterEmptyValidProofs
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
	signers := []common.Address{}
	for _, p := range proofChunks {
		signer, err := chainUtils.RecoverSigner(hash, p)
		if err != nil {
			return nil, err
		}
		signers = append(signers, signer)
	}

	return signers, nil
}

func SplitProofToChunk(proof []byte) ([][]byte, error) {
	if len(proof) == 0 {
		return nil, errorSentinel.ErrReporterEmptyProofParam
	}

	if len(proof)%65 != 0 {
		return nil, errorSentinel.ErrReporterInvalidProofLength
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
