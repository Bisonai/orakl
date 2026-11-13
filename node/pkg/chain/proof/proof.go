// Package proof turns a round's concatenated signatures into the proof
// SubmissionProxy accepts: duplicates dropped, every signer whitelisted, and
// signatures in on-chain whitelist order.
package proof

import (
	"bytes"
	"strings"
	"time"

	chainutils "bisonai.com/miko/node/pkg/chain/utils"
	errorsentinel "bisonai.com/miko/node/pkg/error"
	kaiacommon "github.com/kaiachain/kaia/common"
	"github.com/rs/zerolog/log"
)

const signatureLength = 65

// Order validates proof (concatenated 65-byte signatures over value,
// timestamp and symbol) against whitelist and returns the signatures in
// whitelist order. Any non-whitelisted signer fails with
// ErrDalSignerNotWhitelisted.
func Order(proof []byte, value int64, timestamp time.Time, symbol string, whitelist []kaiacommon.Address) ([]byte, error) {
	proofChunks, err := uniqueChunks(proof)
	if err != nil {
		log.Error().Err(err).Msg("failed to remove duplicate proofs")
		return nil, err
	}

	hash := chainutils.Value2HashForSign(value, timestamp.UnixMilli(), symbol)

	signers, err := signersOf(hash, proofChunks)
	if err != nil {
		log.Error().Err(err).Msg("failed to get signer list from proofs")
		return nil, err
	}

	if err := checkWhitelisted(signers, whitelist); err != nil {
		log.Error().Err(err).Msg("non-whitelisted signers found")
		return nil, err
	}

	signerMap := make(map[kaiacommon.Address][]byte, len(signers))
	for i, signer := range signers {
		signerMap[signer] = proofChunks[i]
	}
	return inWhitelistOrder(signerMap, whitelist)
}

func uniqueChunks(proof []byte) ([][]byte, error) {
	proofs, err := splitChunks(proof)
	if err != nil {
		return nil, err
	}

	unique := make(map[string][]byte)
	for _, p := range proofs {
		unique[string(p)] = p
	}

	result := make([][]byte, 0, len(unique))
	for _, p := range unique {
		result = append(result, p)
	}
	return result, nil
}

func splitChunks(proof []byte) ([][]byte, error) {
	if len(proof) == 0 {
		return nil, errorsentinel.ErrDalEmptyProofParam
	}
	if len(proof)%signatureLength != 0 {
		return nil, errorsentinel.ErrDalInvalidProofLength
	}

	proofs := make([][]byte, 0, len(proof)/signatureLength)
	for i := 0; i < len(proof); i += signatureLength {
		proofs = append(proofs, proof[i:i+signatureLength])
	}
	return proofs, nil
}

func signersOf(hash []byte, proofChunks [][]byte) ([]kaiacommon.Address, error) {
	signers := make([]kaiacommon.Address, 0, len(proofChunks))
	for _, p := range proofChunks {
		signer, err := chainutils.RecoverSigner(hash, p)
		if err != nil {
			return nil, err
		}
		signers = append(signers, signer)
	}
	return signers, nil
}

func checkWhitelisted(signers []kaiacommon.Address, whitelist []kaiacommon.Address) error {
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

func inWhitelistOrder(signerMap map[kaiacommon.Address][]byte, whitelist []kaiacommon.Address) ([]byte, error) {
	ordered := make([][]byte, 0, len(whitelist))
	for _, signer := range whitelist {
		if p, ok := signerMap[signer]; ok {
			ordered = append(ordered, p)
		}
	}

	if len(ordered) == 0 {
		log.Error().Str("Player", "DAL").Msg("no valid proofs")
		return nil, errorsentinel.ErrDalEmptyValidProofs
	}
	return bytes.Join(ordered, nil), nil
}
