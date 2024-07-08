package main

import (
	"context"
	"fmt"
	"math/big"
	"os"

	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/dal/common"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/rs/zerolog/log"
)

const (
	SINGLE_PAIR        = "ADA-USDT"
	SUBMIT_WITH_PROOFS = "submit(bytes32[] calldata _feedHashes, int256[] calldata _answers, uint256[] calldata _timestamps, bytes[] calldata _proofs)"
)

func main() {
	ctx := context.Background()
	url := fmt.Sprintf("http://localhost:8090/api/v1/dal/latest-data-feeds/%s", SINGLE_PAIR)
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

	result, err := request.Request[common.OutgoingSubmissionData](request.WithEndpoint(url))
	if err != nil {
		log.Error().Err(err).Str("Player", "TestConsumer").Msg("failed to get data feed")
		panic(err)
	}

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

	feedHashes := [][32]byte{result.FeedHash}
	values := []*big.Int{&submissionVal}
	timestamps := []*big.Int{&submissionTime}
	proofs := [][]byte{result.Proof}

	rawTx, err := kaiaHelper.MakeDirectTx(ctx, contractAddr, SUBMIT_WITH_PROOFS, feedHashes, values, timestamps, proofs)
	if err != nil {
		log.Error().Err(err).Msg("MakeDirect")
		panic(err)
	}

	log.Debug().Any("feedHashes", feedHashes).Msg("feedHashes")
	log.Debug().Any("values", values).Msg("values")
	log.Debug().Any("timestamps", timestamps).Msg("timestamps")
	log.Debug().Any("proofs", proofs).Msg("proofs")

	log.Debug().Any("tx", rawTx).Msg("tx")

	err = kaiaHelper.SubmitRawTx(ctx, rawTx)
	if err != nil {
		log.Error().Err(err).Msg("SubmitRawTx")
		panic(err)
	}

}
