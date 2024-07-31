package main

import (
	"context"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/dal/common"
	"bisonai.com/orakl/node/pkg/utils/request"
	klaytncommon "github.com/klaytn/klaytn/common"
	"github.com/rs/zerolog/log"
)

const (
	// SINGLE_PAIR        = "ADA-USDT"
	// SUBMIT_WITH_PROOFS = "submit(bytes32[] calldata _feedHashes, int256[] calldata _answers, uint256[] calldata _timestamps, bytes[] calldata _proofs)"
	SUBMIT_STRICT = "submitStrict(bytes32[] calldata _feedHashes, int256[] calldata _answers, uint256[] calldata _timestamps, bytes[] calldata _proofs)"
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

	for i := 0; i < 100; i++ {
		results, err := request.Request[[]common.OutgoingSubmissionData](request.WithEndpoint(url), request.WithHeaders(map[string]string{"X-API-Key": "PBCTNTAfgnGmDRbdzEor"}))
		if err != nil {
			log.Error().Err(err).Str("Player", "TestConsumer").Msg("failed to get data feed")
			panic(err)
		}

		feedHashes := [][32]byte{}
		values := []*big.Int{}
		timestamps := []*big.Int{}
		proofs := [][]byte{}

		for _, entry := range results {
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

			feedHashBytes := klaytncommon.Hex2Bytes(strings.TrimPrefix(entry.FeedHash, "0x"))
			feedHash := [32]byte{}
			copy(feedHash[:], feedHashBytes)

			feedHashes = append(feedHashes, feedHash)
			values = append(values, &submissionVal)
			timestamps = append(timestamps, &submissionTime)
			proofs = append(proofs, klaytncommon.Hex2Bytes(strings.TrimPrefix(entry.Proof, "0x")))

		}
		wg := sync.WaitGroup{}
		for start := 0; start < len(feedHashes); start += 50 {
			end := min(start+50, len(feedHashes))

			batchFeedHashes := feedHashes[start:end]
			batchValues := values[start:end]
			batchTimestamps := timestamps[start:end]
			batchProofs := proofs[start:end]
			wg.Add(1)
			go func() {
				defer wg.Done()
				err = kaiaHelper.SubmitDelegatedFallbackDirect(ctx, contractAddr, SUBMIT_STRICT, batchFeedHashes, batchValues, batchTimestamps, batchProofs)
				if err != nil {
					log.Error().Err(err).Msg("MakeDirect")
				}
			}()
		}
		wg.Wait()

		time.Sleep(15 * time.Second)
	}

}
