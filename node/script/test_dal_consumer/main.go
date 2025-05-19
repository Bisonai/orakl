package main

import (
	"context"
	"math/big"
	"os"
	"strings"

	"bisonai.com/miko/node/pkg/chain/helper"
	"bisonai.com/miko/node/pkg/dal/common"
	"bisonai.com/miko/node/pkg/utils/request"
	kaiacommon "github.com/kaiachain/kaia/common"
	"github.com/rs/zerolog/log"
)

const (
	// SINGLE_PAIR        = "ADA-USDT"
	// SUBMIT_WITH_PROOFS = "submit(bytes32[] calldata _feedHashes, int256[] calldata _answers, uint256[] calldata _timestamps, bytes[] calldata _proofs)"
	SUBMIT_STRICT          = "submitStrict(bytes32[] calldata _feedHashes, int256[] calldata _answers, uint256[] calldata _timestamps, bytes[] calldata _proofs)"
	maxTxSubmissionRetries = 3
)

func main() {
	ctx := context.Background()
	url := "https://dal.baobab.orakl.network/latest-data-feeds/all"
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

	for i := 0; i < 10; i++ {
		results, err := request.Request[[]common.OutgoingSubmissionData](request.WithEndpoint(url), request.WithHeaders(map[string]string{"X-API-Key": ""}))
		if err != nil {
			log.Error().Err(err).Str("Player", "TestConsumer").Msg("failed to get data feed")
			panic(err)
		}

		feedHashes := [][32]byte{}
		values := []*big.Int{}
		timestamps := []*big.Int{}
		proofs := [][]byte{}

		for _, entry := range results {
			log.Info().Any("result", entry).Msg("got data feed")

			var submissionVal big.Int
			_, success := submissionVal.SetString(entry.Value, 10)
			if !success {
				log.Error().Str("Player", "TestConsumer").Msg("failed to convert string to big int")
				panic("failed to convert string to big int")
			}

			var submissionTime big.Int
			_, success = submissionTime.SetString(entry.AggregateTime, 10)
			if !success {
				log.Error().Str("Player", "TestConsumer").Msg("failed to convert string to big int")
				panic("failed to convert string to big int")
			}

			feedHashBytes := kaiacommon.Hex2Bytes(strings.TrimPrefix(entry.FeedHash, "0x"))
			feedHash := [32]byte{}
			copy(feedHash[:], feedHashBytes)

			feedHashes = append(feedHashes, feedHash)
			values = append(values, &submissionVal)
			timestamps = append(timestamps, &submissionTime)
			proofs = append(proofs, kaiacommon.Hex2Bytes(strings.TrimPrefix(entry.Proof, "0x")))

			if len(feedHashes) >= 50 {
				err = kaiaHelper.SubmitDelegatedFallbackDirect(ctx, contractAddr, SUBMIT_STRICT, feedHashes, values, timestamps, proofs)
				if err != nil {
					log.Error().Err(err).Msg("MakeDirect")
					panic(err)
				}

				feedHashes = [][32]byte{}
				values = []*big.Int{}
				timestamps = []*big.Int{}
				proofs = [][]byte{}
			}
		}

		err = kaiaHelper.SubmitDelegatedFallbackDirect(ctx, contractAddr, SUBMIT_STRICT, feedHashes, values, timestamps, proofs)
		if err != nil {
			log.Error().Err(err).Msg("MakeDirect")
			panic(err)
		}
	}
}
