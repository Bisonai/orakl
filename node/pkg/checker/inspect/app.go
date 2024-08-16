package inspect

import (
	"context"
	"errors"
	"math/big"
	"os"

	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/secrets"
)

const (
	DefaultCheckCron = "0 0 * * *"

	RequestVRFSignature = ""
	RequestRRSignature  = ""

	ReadRandomWordSignature = ""
	ReadResponseSignature   = ""
	VRFRequestIdSignature   = ""
	RRRequestIdSignature    = ""
)

func Start(ctx context.Context) error {
	pk := secrets.GetSecret("INSEPCTOR_PK")
	if pk == "" {
		return errors.New("missing INSEPCTOR_PK")
	}

	accountID := os.Getenv("ACC_ID")
	if accountID == "" {
		return errors.New("missing ACC_ID")
	}

	keyHash := os.Getenv("VRF_KEYHASH")
	if keyHash == "" {
		return errors.New("missing VRF_KEYHASH")
	}

	consumerContractAddress := os.Getenv("INSPECT_CONSUMER_ADDRESS")
	if consumerContractAddress == "" {
		return errors.New("missing INSPECT_CONSUMER_ADDRESS")
	}

	chainHelper, err := helper.NewChainHelper(ctx, helper.WithReporterPk(pk), helper.WithoutAdditionalProviderUrls())
	if err != nil {
		return err
	}

}

func requestVrf(ctx context.Context, chainHelper *helper.ChainHelper, address string, signature string) error {
	return nil
}

func requestRR(ctx context.Context, chainHelper *helper.ChainHelper, address string, signature string) error {

}

func readIntValueFromContract(ctx context.Context, chainHelper *helper.ChainHelper, address string, signature string) (*big.Int, error) {
	rawResult, err := chainHelper.ReadContract(ctx, address, signature)
	if err != nil {
		return nil, err
	}

	rawResultSlice, ok := rawResult.([]interface{})
	if !ok {
		return nil, errors.New("failed to parse result")
	}

	return rawResultSlice[0].(*big.Int), nil
}
