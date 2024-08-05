package collector

import (
	"bytes"
	"context"
	"errors"
	"time"

	"bisonai.com/orakl/node/pkg/chain/websocketchainreader"

	chainutils "bisonai.com/orakl/node/pkg/chain/utils"
	errorsentinel "bisonai.com/orakl/node/pkg/error"
	"github.com/klaytn/klaytn/blockchain/types"
	klaytncommon "github.com/klaytn/klaytn/common"
	"github.com/rs/zerolog/log"
)

func getAllOracles(ctx context.Context, chainReader *websocketchainreader.ChainReader, submissionProxyContractAddr string) ([]klaytncommon.Address, error) {
	rawResult, err := chainReader.ReadContractOnce(ctx, websocketchainreader.Kaia, submissionProxyContractAddr, GetAllOracles)
	if err != nil {
		log.Error().Err(err).Msg("failed to get all oracles")
		return nil, err
	}
	rawResultSlice, ok := rawResult.([]interface{})
	if !ok {
		return nil, errors.New("failed to cast result to []interface{} in getAllOracles")
	}

	addresses, ok := rawResultSlice[0].([]klaytncommon.Address)
	if !ok {
		return nil, errors.New("failed to cast first element to []klaytncommon.Address")
	}

	return addresses, nil
}

func subscribeAddOracleEvent(ctx context.Context, chainReader *websocketchainreader.ChainReader, submissionProxyContractAddr string, isUpdated chan any) error {
	logChannel := make(chan types.Log)
	err := chainReader.Subscribe(
		ctx,
		websocketchainreader.WithAddress(submissionProxyContractAddr),
		websocketchainreader.WithChainType(websocketchainreader.Kaia),
		websocketchainreader.WithChannel(logChannel),
	)
	if err != nil {
		return err
	}

	eventName, input, _, eventParseErr := chainutils.ParseMethodSignature(OracleAdded)
	if eventParseErr != nil {
		return eventParseErr
	}

	oracleAddedEventABI, err := chainutils.GenerateEventABI(eventName, input)
	if err != nil {
		return err
	}

	go func() {
		defer close(logChannel)
		for eventLog := range logChannel {
			result, err := oracleAddedEventABI.Unpack(eventName, eventLog.Data)
			if err != nil {
				log.Error().Err(err).Msg("failed to unpack event log data in subscribeAddOracleEvent")
				continue
			}

			_, ok := result[0].(klaytncommon.Address)
			if !ok {
				log.Error().Msg("failed to cast result to klaytncommon.Address in subscribeAddOracleEvent")
				continue
			}

			isUpdated <- true
		}
	}()

	return nil
}

func orderProof(ctx context.Context, proof []byte, value int64, timestamp time.Time, symbol string, cachedWhitelist []klaytncommon.Address) ([]byte, error) {
	proof = removeDuplicateProof(proof)
	hash := chainutils.Value2HashForSign(value, timestamp.UnixMilli(), symbol)
	proofChunks, err := splitProofToChunk(proof)
	if err != nil {
		log.Error().Err(err).Msg("failed to split proof to chunks in orderProof")
		return nil, err
	}

	signers, err := getSignerListFromProofs(hash, proofChunks)
	if err != nil {
		log.Error().Err(err).Msg("failed to get signer list from proofs in orderProof")
		return nil, err
	}

	err = checkForNonWhitelistedSigners(signers, cachedWhitelist)
	if err != nil {
		log.Error().Err(err).Msg("non-whitelisted signers found in orderProof")
		return nil, err
	}

	signerMap := getSignerMap(signers, proofChunks)
	return validateProof(signerMap, cachedWhitelist)
}

func removeDuplicateProof(proof []byte) []byte {
	proofs, err := splitProofToChunk(proof)
	if err != nil {
		return []byte{}
	}

	uniqueProofs := make(map[string][]byte)
	for _, p := range proofs {
		uniqueProofs[string(p)] = p
	}

	result := make([][]byte, 0, len(uniqueProofs))
	for _, p := range uniqueProofs {
		result = append(result, p)
	}

	return bytes.Join(result, nil)
}

func splitProofToChunk(proof []byte) ([][]byte, error) {
	if len(proof) == 0 {
		return nil, errorsentinel.ErrDalEmptyProofParam
	}

	if len(proof)%65 != 0 {
		return nil, errorsentinel.ErrDalInvalidProofLength
	}

	proofs := make([][]byte, 0, len(proof)/65)
	for i := 0; i < len(proof); i += 65 {
		proofs = append(proofs, proof[i:i+65])
	}

	return proofs, nil
}

func getSignerListFromProofs(hash []byte, proofChunks [][]byte) ([]klaytncommon.Address, error) {
	signers := []klaytncommon.Address{}
	for _, p := range proofChunks {
		signer, err := chainutils.RecoverSigner(hash, p)
		if err != nil {
			return nil, err
		}
		signers = append(signers, signer)
	}

	return signers, nil
}

func checkForNonWhitelistedSigners(signers []klaytncommon.Address, whitelist []klaytncommon.Address) error {
	for _, signer := range signers {
		if !isWhitelisted(signer, whitelist) {
			log.Error().Str("Player", "DAL").Str("signer", signer.Hex()).Msg("non-whitelisted signer")
			return errorsentinel.ErrDalSignerNotWhitelisted
		}
	}
	return nil
}

func isWhitelisted(signer klaytncommon.Address, whitelist []klaytncommon.Address) bool {
	for _, w := range whitelist {
		if w == signer {
			return true
		}
	}
	return false
}

func getSignerMap(signers []klaytncommon.Address, proofChunks [][]byte) map[klaytncommon.Address][]byte {
	signerMap := make(map[klaytncommon.Address][]byte)
	for i, signer := range signers {
		signerMap[signer] = proofChunks[i]
	}
	return signerMap
}

func validateProof(signerMap map[klaytncommon.Address][]byte, whitelist []klaytncommon.Address) ([]byte, error) {
	tmpProofs := make([][]byte, 0, len(whitelist))
	for _, signer := range whitelist {
		tmpProof, ok := signerMap[signer]
		if ok {
			tmpProofs = append(tmpProofs, tmpProof)
		}
	}

	if len(tmpProofs) == 0 {
		log.Error().Str("Player", "DAL").Msg("no valid proofs")
		return nil, errorsentinel.ErrDalEmptyValidProofs
	}

	return bytes.Join(tmpProofs, nil), nil
}

func formatBytesToHex(bytes []byte) string {
	return "0x" + klaytncommon.Bytes2Hex(bytes)
}
