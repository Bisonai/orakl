package main

import (
	"context"

	"bisonai.com/orakl/node/pkg/utils"
	"github.com/rs/zerolog/log"
)

func testContractDelegatedCall(ctx context.Context) error {
	txHelper, err := utils.NewTxHelper(ctx)
	if err != nil {
		log.Error().Err(err).Msg("NewTxHelper")
		return err
	}

	rawTx, err := txHelper.MakeFeeDelegatedTx(ctx, "0x93120927379723583c7a0dd2236fcb255e96949f", "increment()")
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

	err := testContractDelegatedCall(ctx)
	if err != nil {
		log.Error().Err(err).Msg("testContractDelegatedCall")
	}
}
