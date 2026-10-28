//nolint:all
package main

import (
	"context"
	"math/big"
	"time"

	"bisonai.com/miko/node/pkg/chain/helper"
	"bisonai.com/miko/node/pkg/nodestate"
	"github.com/rs/zerolog/log"
)

const maxTxSubmissionRetries = 3

func testContractFeeDelegatedCall(ctx context.Context, contractAddress string, contractFunction string, args ...interface{}) error {
	kaiaHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		log.Error().Err(err).Msg("NewTxHelper")
		return err
	}

	return kaiaHelper.SubmitDelegatedFallbackDirect(ctx, contractAddress, contractFunction, args...)
}

func main() {
	ctx := context.Background()

	state, err := nodestate.OpenFromEnv(ctx, false)
	if err != nil {
		log.Error().Err(err).Msg("OpenFromEnv")
		return
	}
	defer state.Close()

	s, err := helper.NewSigner(ctx, helper.WithSignerStore(state))
	if err != nil {
		log.Error().Err(err).Msg("NewSigner")

	}

	contractAddress := "0x08f43BebA1B0642C14493C70268a5AC8f380476b"
	contractFunction := `test(int256 _answer, bytes memory _proof)`
	answer := big.NewInt(200000000)
	proof, err := s.MakeGlobalAggregateProof(200000000, time.Now(), "test-aggregate")
	if err != nil {
		log.Error().Err(err).Msg("MakeGlobalAggregateProof")
	}
	proofs := [][]byte{proof, proof}
	testProof := concatBytes(proofs)

	err = testContractFeeDelegatedCall(ctx, contractAddress, contractFunction, answer, testProof)
	if err != nil {
		log.Error().Err(err).Msg("testContractDirectCall")
	}
}

func concatBytes(slices [][]byte) []byte {
	var result []byte
	for _, slice := range slices {
		result = append(result, slice...)
	}
	return result
}
