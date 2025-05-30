package collector

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/chain/chainreader"

	chainutils "bisonai.com/miko/node/pkg/chain/utils"
	errorsentinel "bisonai.com/miko/node/pkg/error"
	kaiacommon "github.com/kaiachain/kaia/common"
	"github.com/rs/zerolog/log"
)

func getAllOracles(ctx context.Context, chainReader *chainreader.ChainReader, submissionProxyContractAddr string) ([]kaiacommon.Address, error) {
	rawResult, err := chainReader.ReadContract(ctx, submissionProxyContractAddr, GetAllOracles)
	if err != nil {
		log.Error().Err(err).Msg("failed to get all oracles")
		return nil, err
	}
	rawResultSlice, ok := rawResult.([]interface{})
	if !ok {
		return nil, errors.New("failed to cast result to []interface{} in getAllOracles")
	}

	addresses, ok := rawResultSlice[0].([]kaiacommon.Address)
	if !ok {
		return nil, errors.New("failed to cast first element to []kaiacommon.Address")
	}

	log.Info().Any("addresses", addresses).Msg("loaded oracles")

	return addresses, nil
}

func orderProof(ctx context.Context, proof []byte, value int64, timestamp time.Time, symbol string, cachedWhitelist []kaiacommon.Address) ([]byte, error) {
	proofChunks, err := getUniqueProofChunks(proof)
	if err != nil {
		log.Error().Err(err).Msg("failed to remove duplicate proofs in orderProof")
		return nil, err
	}

	hash := chainutils.Value2HashForSign(value, timestamp.UnixMilli(), symbol)

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

func getUniqueProofChunks(proof []byte) ([][]byte, error) {
	proofs, err := splitProofToChunk(proof)
	if err != nil {
		return nil, err
	}

	uniqueProofs := make(map[string][]byte)
	for _, p := range proofs {
		uniqueProofs[string(p)] = p
	}

	result := make([][]byte, 0, len(uniqueProofs))
	for _, p := range uniqueProofs {
		result = append(result, p)
	}

	return result, nil
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

func getSignerListFromProofs(hash []byte, proofChunks [][]byte) ([]kaiacommon.Address, error) {
	signers := []kaiacommon.Address{}
	for _, p := range proofChunks {
		signer, err := chainutils.RecoverSigner(hash, p)
		if err != nil {
			return nil, err
		}
		signers = append(signers, signer)
	}

	return signers, nil
}

func checkForNonWhitelistedSigners(signers []kaiacommon.Address, whitelist []kaiacommon.Address) error {
	for _, signer := range signers {
		if !isWhitelisted(signer, whitelist) {
			log.Error().Str("Player", "DAL").Any("whitelist", whitelist).Any("signer", signer).Msg("non-whitelisted signer")
			return errorsentinel.ErrDalSignerNotWhitelisted
		}
	}
	return nil
}

func isWhitelisted(signer kaiacommon.Address, whitelist []kaiacommon.Address) bool {
	signerHex := strings.TrimSpace(signer.Hex())
	for _, w := range whitelist {
		if strings.EqualFold(strings.TrimSpace(w.Hex()), signerHex) {
			return true
		}
	}
	return false
}

func getSignerMap(signers []kaiacommon.Address, proofChunks [][]byte) map[kaiacommon.Address][]byte {
	signerMap := make(map[kaiacommon.Address][]byte)
	for i, signer := range signers {
		signerMap[signer] = proofChunks[i]
	}
	return signerMap
}

func validateProof(signerMap map[kaiacommon.Address][]byte, whitelist []kaiacommon.Address) ([]byte, error) {
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
	return "0x" + kaiacommon.Bytes2Hex(bytes)
}
