package main

import (
	"context"
	"math/big"
	"time"

	"bisonai.com/orakl/node/pkg/chain/helper"
	"github.com/rs/zerolog/log"
)

func testContractDirectCall(ctx context.Context, contractAddress string, contractFunction string, args ...interface{}) error {
	klaytnHelper, err := helper.NewKlayHelper(ctx, "")
	if err != nil {
		log.Error().Err(err).Msg("NewTxHelper")
		return err
	}

	rawTx, err := klaytnHelper.MakeDirectTx(ctx, contractAddress, contractFunction, args...)
	if err != nil {
		log.Error().Err(err).Msg("MakeDirect")
		return err
	}

	return klaytnHelper.SubmitRawTx(ctx, rawTx)
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
	proof, err := s.MakeGlobalAggregateProof(200000000, time.Now())
	if err != nil {
		log.Error().Err(err).Msg("MakeGlobalAggregateProof")
	}
	proofs := [][]byte{proof, proof}
	testProof := concatBytes(proofs)

	err = testContractDirectCall(ctx, contractAddress, contractFunction, answer, testProof)
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
