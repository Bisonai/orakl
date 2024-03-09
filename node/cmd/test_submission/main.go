package main

import (
	"context"

	"bisonai.com/orakl/node/pkg/utils"
)

// func testContractDirectCall(ctx context.Context) error {
// 	rawTx, err := utils.TestMakeRawTxV2(ctx)
// 	if err != nil {
// 		return err
// 	}

// 	log.Info().Msgf("Raw transaction: %s", rawTx)

// 	err = utils.TestSendRawTx(ctx, rawTx)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

func testContractDelegatedCall(ctx context.Context) error {
	rawTx, err := utils.TestMakeFeeDelegatedRawTx(ctx)
	if err != nil {
		return err
	}

	signedRawTx, err := utils.SignTxByFeePayer(ctx, rawTx)
	if err != nil {
		return err
	}

	signedRawTxHash, err := utils.GetRawTxHash(signedRawTx)
	if err != nil {
		return err
	}

	return utils.TestSendRawTx(ctx, signedRawTxHash)

}

func main() {
	ctx := context.Background()
	testContractDelegatedCall(ctx)
}
