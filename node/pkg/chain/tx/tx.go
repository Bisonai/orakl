package tx

import (
	"context"
	"errors"
	"math/big"
	"os"
	"strings"

	chain_utils "bisonai.com/orakl/node/pkg/chain/utils"
	"bisonai.com/orakl/node/pkg/utils/request"

	"github.com/klaytn/klaytn/blockchain/types"
	"github.com/klaytn/klaytn/client"

	"github.com/rs/zerolog/log"
)

type TxHelper struct {
	client       *client.Client
	wallets      []string
	chainID      *big.Int
	delegatorUrl string

	lastUsed int
}

type signedTx struct {
	SignedRawTx *string `json:"signedRawTx" db:"signedRawTx"`
}

func NewTxHelper(ctx context.Context) (*TxHelper, error) {
	wallets, err := chain_utils.GetWallets(ctx)
	if err != nil {
		return nil, err
	}

	if os.Getenv("REPORTER_PK") != "" {
		wallet := strings.TrimPrefix(os.Getenv("REPORTER_PK"), "0x")
		wallets = append(wallets, wallet)
	}

	delegatorUrl := os.Getenv("DELEGATOR_URL")
	providerUrl := os.Getenv("PROVIDER_URL")

	client, err := client.Dial(providerUrl)
	if err != nil {
		return nil, err
	}

	chainID, err := chain_utils.GetChainID(ctx, client)
	if err != nil {
		return nil, err
	}

	return &TxHelper{
		client:       client,
		wallets:      wallets,
		chainID:      chainID,
		delegatorUrl: delegatorUrl,
	}, nil
}

func (t *TxHelper) Close() {
	t.client.Close()
}

func (t *TxHelper) GetSignedFromDelegator(tx *types.Transaction) (*types.Transaction, error) {
	if t.delegatorUrl == "" {
		return nil, errors.New("delegator url not set")
	}

	payload, err := chain_utils.MakePayload(tx)
	if err != nil {
		return nil, err
	}

	result, err := request.UrlRequest[signedTx](t.delegatorUrl+"/api/v1/sign/volatile", "POST", payload, nil, "")
	if err != nil {
		log.Error().Err(err).Msg("failed to request sign from delegator")
		return nil, err
	}

	if result.SignedRawTx == nil {
		return nil, errors.New("no signed raw tx")
	}
	return chain_utils.HashToTx(*result.SignedRawTx)
}

func (t *TxHelper) NextReporter() string {
	if len(t.wallets) == 0 {
		return ""
	}
	reporter := t.wallets[t.lastUsed]
	t.lastUsed = (t.lastUsed + 1) % len(t.wallets)
	return reporter
}

func (t *TxHelper) MakeDirectTx(ctx context.Context, contractAddressHex string, functionString string, args ...interface{}) (*types.Transaction, error) {
	return chain_utils.MakeDirectTx(ctx, t.client, contractAddressHex, t.NextReporter(), functionString, t.chainID, args...)
}

func (t *TxHelper) MakeFeeDelegatedTx(ctx context.Context, contractAddressHex string, functionString string, args ...interface{}) (*types.Transaction, error) {
	return chain_utils.MakeFeeDelegatedTx(ctx, t.client, contractAddressHex, t.NextReporter(), functionString, t.chainID, args...)
}

func (t *TxHelper) SubmitRawTx(ctx context.Context, tx *types.Transaction) error {
	return chain_utils.SubmitRawTx(ctx, t.client, tx)
}

func (t *TxHelper) SubmitRawTxString(ctx context.Context, rawTx string) error {
	return chain_utils.SubmitRawTxString(ctx, t.client, rawTx)
}

func (t *TxHelper) SignTxByFeePayer(ctx context.Context, tx *types.Transaction) (*types.Transaction, error) {
	return chain_utils.SignTxByFeePayer(ctx, t.client, tx, t.chainID)
}
