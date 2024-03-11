package main

import (
	"context"
	"flag"

	"bisonai.com/orakl/node/pkg/utils"
	"github.com/rs/zerolog/log"
)

func testContractDelegatedCall(ctx context.Context, contractAddress string, contractFunction string, args ...interface{}) error {
	txHelper, err := utils.NewTxHelper(ctx)
	if err != nil {
		log.Error().Err(err).Msg("NewTxHelper")
		return err
	}

	rawTx, err := txHelper.MakeFeeDelegatedTx(ctx, contractAddress, contractFunction, args...)
	if err != nil {
		log.Error().Err(err).Msg("MakeFeeDelegatedTx")
		return err
	}

	signedRawTx, err := txHelper.SignTxByFeePayer(ctx, rawTx)
	if err != nil {
		log.Error().Err(err).Msg("SignTxByFeePayer")
		return err
	}

	return txHelper.SubmitRawTx(ctx, signedRawTx)
}

func main() {
	ctx := context.Background()
	contractAddress := flag.String("c", "0x93120927379723583c7a0dd2236fcb255e96949f", "contract address")
	contractFunction := flag.String("f", "increment()", "contract function")

	flag.Parse()

	log.Info().Msgf("contractAddress: %s", *contractAddress)
	log.Info().Msgf("contractFunction: %s", *contractFunction)

	err := testContractDelegatedCall(ctx, *contractAddress, *contractFunction)
	if err != nil {
		log.Error().Err(err).Msg("testContractDelegatedCall")
	}

	// example code for dummy batch submission, check args usage from the code below

	// contractAddress := "0x8fb610c0Cc27Ca7726fad4c8696d09ca0E8eAee1"
	// contractFunction := `batchSubmit(string[] memory _pairs, int256[] memory _prices)`
	// pairs := []string{"BTC-USD", "ETH-USD"}
	// prices := []*big.Int{big.NewInt(100000000), big.NewInt(200000000)}
	// err := testContractDelegatedCall(ctx, contractAddress, contractFunction, pairs, prices)
}
