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
	clients      []utils.ClientInterface
	wallets      []string
	chainID      *big.Int
	delegatorUrl string

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

	primaryClient, err := eth_client.Dial(providerUrl)
	if err != nil {
		return nil, err
	}

	chainID, err := utils.GetChainID(ctx, primaryClient)
	if err != nil {
		log.Error().Err(err).Msg("failed to get chain id based on:" + providerUrl)
		return nil, err
	}
	providerUrls, err := utils.LoadProviderUrls(ctx, int(chainID.Int64()))
	if err != nil {
		log.Error().Err(err).Msg("failed to load provider urls")
		return nil, err
	}
	clients := make([]utils.ClientInterface, len(providerUrls)+1)
	clients[0] = primaryClient
	for _, url := range providerUrls {
		subClient, err := eth_client.Dial(url)
		if err != nil {
			log.Error().Err(err).Msg("failed to dial sub client")
			continue
		}
		clients = append(clients, subClient)
	}

	return newHelper(ctx, clients, reporterPk, chainID)
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

	primaryClient, err := client.Dial(providerUrl)
	if err != nil {
		return nil, err
	}
	chainID, err := utils.GetChainID(ctx, primaryClient)
	if err != nil {
		log.Error().Err(err).Msg("failed to get chain id based on:" + providerUrl)
		return nil, err
	}
	providerUrls, err := utils.LoadProviderUrls(ctx, int(chainID.Int64()))
	if err != nil {
		log.Error().Err(err).Msg("failed to load provider urls")
		return nil, err
	}
	clients := make([]utils.ClientInterface, len(providerUrls)+1)
	clients[0] = primaryClient
	for _, url := range providerUrls {
		subClient, err := client.Dial(url)
		if err != nil {
			log.Error().Err(err).Msg("failed to dial sub client")
			continue
		}
		clients = append(clients, subClient)
	}

	return newHelper(ctx, clients, reporterPk, chainID)
}

func newHelper(ctx context.Context, clients []utils.ClientInterface, reporterPK string, chainID *big.Int) (*ChainHelper, error) {
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

	return &ChainHelper{
		clients:      clients,
		wallets:      wallets,
		chainID:      chainID,
		delegatorUrl: delegatorUrl,
	}, nil
}

func (t *ChainHelper) Close() {
	for _, helperClient := range t.clients {
		helperClient.Close()
	}
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
	var result *types.Transaction
	job := func(c utils.ClientInterface) error {
		tmp, err := utils.MakeDirectTx(ctx, c, contractAddressHex, t.NextReporter(), functionString, t.chainID, args...)
		if err == nil {
			result = tmp
		}
		return err
	}
	err := t.retryOnJsonRpcFailure(ctx, job)
	return result, err
}

func (t *ChainHelper) MakeFeeDelegatedTx(ctx context.Context, contractAddressHex string, functionString string, args ...interface{}) (*types.Transaction, error) {
	var result *types.Transaction
	job := func(c utils.ClientInterface) error {
		tmp, err := utils.MakeFeeDelegatedTx(ctx, c, contractAddressHex, t.NextReporter(), functionString, t.chainID, args...)
		if err == nil {
			result = tmp
		}
		return err
	}
	err := t.retryOnJsonRpcFailure(ctx, job)
	return result, err
}

func (t *ChainHelper) SubmitRawTx(ctx context.Context, tx *types.Transaction) error {
	job := func(c utils.ClientInterface) error {
		return utils.SubmitRawTx(ctx, c, tx)
	}
	return t.retryOnJsonRpcFailure(ctx, job)
}

func (t *ChainHelper) SubmitRawTxString(ctx context.Context, rawTx string) error {
	job := func(c utils.ClientInterface) error {
		return utils.SubmitRawTxString(ctx, c, rawTx)
	}
	return t.retryOnJsonRpcFailure(ctx, job)
}

// SignTxByFeePayer: used for testing purpose
func (t *ChainHelper) SignTxByFeePayer(ctx context.Context, tx *types.Transaction) (*types.Transaction, error) {
	var result *types.Transaction
	job := func(c utils.ClientInterface) error {
		tmp, err := utils.SignTxByFeePayer(ctx, c, tx, t.chainID)
		if err == nil {
			result = tmp
		}
		return err
	}
	err := t.retryOnJsonRpcFailure(ctx, job)
	return result, err
}

func (t *ChainHelper) ReadContract(ctx context.Context, contractAddressHex string, functionString string, args ...interface{}) (interface{}, error) {
	var result interface{}
	job := func(c utils.ClientInterface) error {
		tmp, err := utils.ReadContract(ctx, c, functionString, contractAddressHex, args...)
		if err == nil {
			result = tmp
		}
		return err
	}
	err := t.retryOnJsonRpcFailure(ctx, job)
	return result, err
}

func (t *ChainHelper) ChainID() *big.Int {
	return t.chainID
}

func (t *ChainHelper) retryOnJsonRpcFailure(ctx context.Context, job func(c utils.ClientInterface) error) error {
	for _, client := range t.clients {
		err := job(client)
		if err != nil {
			if utils.ShouldRetryWithSwitchedJsonRPC(err) {
				continue
			}
			return err
		}
		break
	}
	return nil
}
