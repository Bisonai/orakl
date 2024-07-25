package main

import (
	"context"
	"fmt"
	"math/big"
	"os"

	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/dal/common"
	"bisonai.com/orakl/node/pkg/utils/request"
	klaytncommon "github.com/klaytn/klaytn/common"
	"github.com/rs/zerolog/log"
)

const (
	SINGLE_PAIR        = "ADA-USDT"
	SUBMIT_WITH_PROOFS = "submit(bytes32[] calldata _feedHashes, int256[] calldata _answers, uint256[] calldata _timestamps, bytes[] calldata _proofs)"
)

func main() {
	ctx := context.Background()
	url := fmt.Sprintf("http://localhost:8090/latest-data-feeds/%s", SINGLE_PAIR)
	contractAddr := os.Getenv("SUBMISSION_PROXY_CONTRACT")
	if contractAddr == "" {
		log.Error().Msg("Missing SUBMISSION_PROXY_CONTRACT")
		panic("Missing SUBMISSION_PROXY_CONTRACT")
	}

	kaiaHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		log.Error().Err(err).Msg("NewTxHelper")
		panic(err)
	}

	results, err := request.Request[[]common.OutgoingSubmissionData](request.WithEndpoint(url), request.WithHeaders(map[string]string{"X-API-Key": "testkey"}))
	if err != nil {
		log.Error().Err(err).Str("Player", "TestConsumer").Msg("failed to get data feed")
		panic(err)
	}

	result := results[0]

	var submissionVal big.Int
	_, success := submissionVal.SetString(result.Value, 10)
	if !success {
		log.Error().Str("Player", "TestConsumer").Msg("failed to convert string to big int")
		panic("failed to convert string to big int")
	}

	var submissionTime big.Int
	_, success = submissionTime.SetString(result.AggregateTime, 10)
	if !success {
		log.Error().Str("Player", "TestConsumer").Msg("failed to convert string to big int")
		panic("failed to convert string to big int")
	}

	feedHashBytes := klaytncommon.Hex2Bytes(result.FeedHash)
	feedHash := [32]byte{}
	copy(feedHash[:], feedHashBytes)

	feedHashes := [][32]byte{feedHash}
	values := []*big.Int{&submissionVal}
	timestamps := []*big.Int{&submissionTime}
	proofs := [][]byte{klaytncommon.Hex2Bytes(result.Proof)}

	err = kaiaHelper.SubmitDelegatedFallbackDirect(ctx, contractAddr, SUBMIT_WITH_PROOFS, feedHashes, values, timestamps, proofs)
	if err != nil {
		log.Error().Err(err).Msg("MakeDirect")
		panic(err)
	}
}
