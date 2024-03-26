package helper

import (
	"context"
	"errors"
	"math/big"
	"os"
	"strings"

	klaytn_utils "bisonai.com/orakl/node/pkg/chain/klaytn/utils"
	"bisonai.com/orakl/node/pkg/utils/request"

	"github.com/klaytn/klaytn/blockchain/types"
	"github.com/klaytn/klaytn/client"

	"github.com/rs/zerolog/log"
)

type KlaytnHelper struct {
	client       *client.Client
	wallets      []string
	chainID      *big.Int
	delegatorUrl string

	lastUsed int
}

type signedTx struct {
	SignedRawTx *string `json:"signedRawTx" db:"signedRawTx"`
}

const (
	DelegatorEndpoint = "/api/v1/sign/volatile"

	// TODO: support multiple json rpc providers
	EnvDelegatorUrl = "DELEGATOR_URL"
	EnvProviderUrl  = "KLAYTN_PROVIDER_URL"
	EnvReporterPk   = "KLAYTN_REPORTER_PK"
)

func NewKlaytnHelper(ctx context.Context) (*KlaytnHelper, error) {
	wallets, err := klaytn_utils.GetWallets(ctx)
	if err != nil {
		return nil, err
	}

	if os.Getenv(EnvReporterPk) != "" {
		wallet := strings.TrimPrefix(os.Getenv(EnvReporterPk), "0x")
		wallets = append(wallets, wallet)
	}

	delegatorUrl := os.Getenv(EnvDelegatorUrl)

	providerUrl := os.Getenv(EnvProviderUrl)
	if providerUrl == "" {
		log.Error().Msg("provider url not set")
		return nil, errors.New("provider url not set")
	}

	client, err := client.Dial(providerUrl)
	if err != nil {
		return nil, err
	}

	chainID, err := klaytn_utils.GetChainID(ctx, client)
	if err != nil {
		return nil, err
	}

	return &KlaytnHelper{
		client:       client,
		wallets:      wallets,
		chainID:      chainID,
		delegatorUrl: delegatorUrl,
	}, nil
}

func (t *KlaytnHelper) Close() {
	t.client.Close()
}

func (t *KlaytnHelper) GetSignedFromDelegator(tx *types.Transaction) (*types.Transaction, error) {
	if t.delegatorUrl == "" {
		return nil, errors.New("delegator url not set")
	}

	payload, err := klaytn_utils.MakePayload(tx)
	if err != nil {
		return nil, err
	}

	result, err := request.UrlRequest[signedTx](t.delegatorUrl+DelegatorEndpoint, "POST", payload, nil, "")
	if err != nil {
		log.Error().Err(err).Msg("failed to request sign from delegator")
		return nil, err
	}

	if result.SignedRawTx == nil {
		return nil, errors.New("no signed raw tx")
	}
	return klaytn_utils.HashToTx(*result.SignedRawTx)
}

func (t *KlaytnHelper) NextReporter() string {
	if len(t.wallets) == 0 {
		return ""
	}
	reporter := t.wallets[t.lastUsed]
	t.lastUsed = (t.lastUsed + 1) % len(t.wallets)
	return reporter
}

func (t *KlaytnHelper) MakeDirectTx(ctx context.Context, contractAddressHex string, functionString string, args ...interface{}) (*types.Transaction, error) {
	return klaytn_utils.MakeDirectTx(ctx, t.client, contractAddressHex, t.NextReporter(), functionString, t.chainID, args...)
}

func (t *KlaytnHelper) MakeFeeDelegatedTx(ctx context.Context, contractAddressHex string, functionString string, args ...interface{}) (*types.Transaction, error) {
	return klaytn_utils.MakeFeeDelegatedTx(ctx, t.client, contractAddressHex, t.NextReporter(), functionString, t.chainID, args...)
}

func (t *KlaytnHelper) SubmitRawTx(ctx context.Context, tx *types.Transaction) error {
	return klaytn_utils.SubmitRawTx(ctx, t.client, tx)
}

func (t *KlaytnHelper) SubmitRawTxString(ctx context.Context, rawTx string) error {
	return klaytn_utils.SubmitRawTxString(ctx, t.client, rawTx)
}

// SignTxByFeePayer: used for testing purpose
func (t *KlaytnHelper) SignTxByFeePayer(ctx context.Context, tx *types.Transaction) (*types.Transaction, error) {
	return klaytn_utils.SignTxByFeePayer(ctx, t.client, tx, t.chainID)
}

// * `functionString` should include `returns()` clause
func (t *KlaytnHelper) ReadContract(ctx context.Context, contractAddressHex string, functionString string, args ...interface{}) (interface{}, error) {
	return klaytn_utils.ReadContract(ctx, t.client, functionString, contractAddressHex, args...)
}
