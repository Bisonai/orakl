//nolint:all
package main

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"bisonai.com/orakl/node/pkg/chain/helper"
	"github.com/rs/zerolog/log"
)

func testContractDirectCall(ctx context.Context, contractAddress string, contractFunction string, args ...interface{}) error {
	klaytnHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		log.Error().Err(err).Msg("NewTxHelper")
		return err
	}

	rawTx, err := klaytnHelper.MakeDirectTx(ctx, contractAddress, contractFunction, args...)
	if err != nil {
		log.Error().Err(err).Msg("MakeDirect")
		return err
	}

	fmt.Println(rawTx.GasPrice().String())

	return klaytnHelper.SubmitRawTx(ctx, rawTx)
}

func testContractFeeDelegatedCall(ctx context.Context, contractAddress string, contractFunction string, args ...interface{}) error {
	klaytnHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		log.Error().Err(err).Msg("NewTxHelper")
		return err
	}
	rawTx, err := klaytnHelper.MakeFeeDelegatedTx(ctx, contractAddress, contractFunction, 1.5, args...)
	if err != nil {
		log.Error().Err(err).Msg("MakeFeeDelegated")
		return err
	}

	signedTx, err := klaytnHelper.SignTxByFeePayer(ctx, rawTx)
	if err != nil {
		log.Error().Err(err).Msg("SignTxByFeePayer")
		return err
	}

	return klaytnHelper.SubmitRawTx(ctx, signedTx)
}

func main() {
	ctx := context.Background()

	s, err := helper.NewSignHelper("")
	if err != nil {
		log.Error().Err(err).Msg("NewSignHelper")

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
