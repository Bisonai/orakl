package helper

import (
	"context"
	"errors"
	"math/big"
	"os"
	"strings"

	"bisonai.com/orakl/node/pkg/chain/eth_client"
	"bisonai.com/orakl/node/pkg/chain/utils"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/klaytn/klaytn/blockchain/types"
	"github.com/klaytn/klaytn/client"
	"github.com/rs/zerolog/log"
)

type ChainHelper struct {
	client       utils.ClientInterface
	wallets      []string
	chainID      *big.Int
	delegatorUrl string
	providerUrls []string // array index stands for priority

	lastUsedWalletIndex int
}

type signedTx struct {
	SignedRawTx *string `json:"signedRawTx" db:"signedRawTx"`
}

const (
	DelegatorEndpoint = "/api/v1/sign/volatile"

	// TODO: support multiple json rpc providers
	EnvDelegatorUrl   = "DELEGATOR_URL"
	KlaytnProviderUrl = "KLAYTN_PROVIDER_URL"
	KlaytnReporterPk  = "KLAYTN_REPORTER_PK"
	EthProviderUrl    = "ETH_PROVIDER_URL"
	EthReporterPk     = "ETH_REPORTER_PK"
)

func NewEthHelper(ctx context.Context, providerUrl string) (*ChainHelper, error) {
	if providerUrl == "" {
		providerUrl = os.Getenv(EthProviderUrl)
		if providerUrl == "" {
			log.Error().Msg("provider url not set")
			return nil, errors.New("provider url not set")
		}
	}

	reporterPk := os.Getenv(EthReporterPk)

	client, err := eth_client.Dial(providerUrl)
	if err != nil {
		return nil, err
	}
	return newHelper(ctx, client, reporterPk)
}

func NewKlayHelper(ctx context.Context, providerUrl string) (*ChainHelper, error) {
	if providerUrl == "" {
		providerUrl = os.Getenv(KlaytnProviderUrl)
		if providerUrl == "" {
			log.Error().Msg("provider url not set")
			return nil, errors.New("provider url not set")
		}
	}

	reporterPk := os.Getenv(KlaytnReporterPk)

	client, err := client.Dial(providerUrl)
	if err != nil {
		return nil, err
	}
	return newHelper(ctx, client, reporterPk)
}

func newHelper(ctx context.Context, client utils.ClientInterface, reporterPK string) (*ChainHelper, error) {
	// assumes that single application submits to single chain, get wallets will select all from wallets table
	wallets, err := utils.GetWallets(ctx)
	if err != nil {
		return nil, err
	}

	if reporterPK != "" {
		wallet := strings.TrimPrefix(reporterPK, "0x")
		wallets = append(wallets, wallet)
	}

	delegatorUrl := os.Getenv(EnvDelegatorUrl)

	providerUrl := os.Getenv(KlaytnProviderUrl)
	if providerUrl == "" {
		log.Error().Msg("provider url not set")
		return nil, errors.New("provider url not set")
	}

	chainID, err := utils.GetChainID(ctx, client)
	if err != nil {
		log.Error().Err(err).Msg("failed to get chain id based on:" + providerUrl)
		return nil, err
	}

	loadedProviderUrls, err := utils.LoadProviderUrls(ctx, int(chainID.Int64()))
	if err != nil {
		log.Error().Err(err).Msg("failed to load provider urls")
		return nil, err

	}
	providerUrls := make([]string, len(loadedProviderUrls)+1)
	providerUrls[0] = providerUrl
	for i, url := range loadedProviderUrls {
		providerUrls[i+1] = url.Url
	}

	return &ChainHelper{
		client:       client,
		wallets:      wallets,
		chainID:      chainID,
		delegatorUrl: delegatorUrl,
		providerUrls: providerUrls,
	}, nil
}

func (t *ChainHelper) Close() {
	t.client.Close()
}

func (t *ChainHelper) GetSignedFromDelegator(tx *types.Transaction) (*types.Transaction, error) {
	if t.delegatorUrl == "" {
		return nil, errors.New("delegator url not set")
	}

	payload, err := utils.MakePayload(tx)
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
	return utils.HashToTx(*result.SignedRawTx)
}

func (t *ChainHelper) NextReporter() string {
	if len(t.wallets) == 0 {
		return ""
	}
	reporter := t.wallets[t.lastUsedWalletIndex]
	t.lastUsedWalletIndex = (t.lastUsedWalletIndex + 1) % len(t.wallets)
	return reporter
}

func (t *ChainHelper) MakeDirectTx(ctx context.Context, contractAddressHex string, functionString string, args ...interface{}) (*types.Transaction, error) {
	return utils.MakeDirectTx(ctx, t.client, contractAddressHex, t.NextReporter(), functionString, t.chainID, args...)
}

func (t *ChainHelper) MakeFeeDelegatedTx(ctx context.Context, contractAddressHex string, functionString string, args ...interface{}) (*types.Transaction, error) {
	return utils.MakeFeeDelegatedTx(ctx, t.client, contractAddressHex, t.NextReporter(), functionString, t.chainID, args...)
}

func (t *ChainHelper) SubmitRawTx(ctx context.Context, tx *types.Transaction) error {
	return utils.SubmitRawTx(ctx, t.client, tx)
}

func (t *ChainHelper) SubmitRawTxString(ctx context.Context, rawTx string) error {
	return utils.SubmitRawTxString(ctx, t.client, rawTx)
}

// SignTxByFeePayer: used for testing purpose
func (t *ChainHelper) SignTxByFeePayer(ctx context.Context, tx *types.Transaction) (*types.Transaction, error) {
	return utils.SignTxByFeePayer(ctx, t.client, tx, t.chainID)
}

func (t *ChainHelper) ReadContract(ctx context.Context, contractAddressHex string, functionString string, args ...interface{}) (interface{}, error) {
	return utils.ReadContract(ctx, t.client, functionString, contractAddressHex, args...)
}

func (t *ChainHelper) ChainID() *big.Int {
	return t.chainID
}
