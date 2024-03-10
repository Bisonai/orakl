package main

import (
	"context"
	"flag"
	"strings"

	"bisonai.com/orakl/node/pkg/utils"
	"github.com/rs/zerolog/log"
)

func testContractDelegatedCall(ctx context.Context, contractAddress string, contractFunction string, args ...string) error {
	txHelper, err := utils.NewTxHelper(ctx)
	if err != nil {
		log.Error().Err(err).Msg("NewTxHelper")
		return err
	}

	rawTx, err := txHelper.MakeFeeDelegatedTx(ctx, contractAddress, contractFunction)
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
	contractArguments := flag.String("a", "", "contract arguments, comma-separated")
	flag.Parse()

	log.Info().Msgf("contractAddress: %s", *contractAddress)
	log.Info().Msgf("contractFunction: %s", *contractFunction)
	log.Info().Msgf("contractArguments: %s", *contractArguments)

	// Split the contractArguments string into arguments
	args := strings.Split(*contractArguments, ",")

	err := testContractDelegatedCall(ctx, *contractAddress, *contractFunction, args...)
	if err != nil {
		log.Error().Err(err).Msg("testContractDelegatedCall")
	}
}
