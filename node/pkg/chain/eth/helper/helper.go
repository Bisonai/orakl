package helper

import (
	"context"
	"errors"
	"math/big"
	"os"
	"strings"

	eth_utils "bisonai.com/orakl/node/pkg/chain/eth/utils"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/rs/zerolog/log"
)

type EthHelper struct {
	client   *ethclient.Client
	wallets  []string
	chainID  *big.Int
	lastUsed int
}

const (
	EnvProviderUrl = "ETH_PROVIDER_URL"
	EnvReporterPk  = "ETH_REPORTER_PK"
)

func NewEthHelper(ctx context.Context) (*EthHelper, error) {
	wallets, err := eth_utils.GetWallets(ctx)
	if err != nil {
		return nil, err
	}

	if os.Getenv(EnvReporterPk) != "" {
		wallet := strings.TrimPrefix(os.Getenv(EnvReporterPk), "0x")
		wallets = append(wallets, wallet)
	}

	providerUrl := os.Getenv(EnvProviderUrl)
	if providerUrl == "" {
		log.Error().Msg("provider url not set")
		return nil, errors.New("provider url not set")
	}

	client, err := ethclient.Dial(providerUrl)
	if err != nil {
		return nil, err
	}

	chainID, err := eth_utils.GetChainID(ctx, client)
	if err != nil {
		return nil, err
	}

	return &EthHelper{
		client:  client,
		wallets: wallets,
		chainID: chainID,
	}, nil
}

func (e *EthHelper) Close() {
	e.client.Close()
}

func (e *EthHelper) NextReporter() string {
	if len(e.wallets) == 0 {
		return ""
	}
	reporter := e.wallets[e.lastUsed]
	e.lastUsed = (e.lastUsed + 1) % len(e.wallets)
	return reporter
}

func (e *EthHelper) MakeDirectTx(ctx context.Context, contractAddressHex string, functionString string, args ...interface{}) (*types.Transaction, error) {
	return eth_utils.MakeDirectTx(ctx, e.client, contractAddressHex, e.NextReporter(), functionString, e.chainID, args...)
}

func (e *EthHelper) SubmitRawTx(ctx context.Context, tx *types.Transaction) error {
	return eth_utils.SubmitRawTx(ctx, e.client, tx)
}

func (e *EthHelper) SubmitRawTxString(ctx context.Context, rawTx string) error {
	return eth_utils.SubmitRawTxString(ctx, e.client, rawTx)
}

func (e *EthHelper) ReadContract(ctx context.Context, contractAddressHex string, functionString string, args ...interface{}) (interface{}, error) {
	return eth_utils.ReadContract(ctx, e.client, functionString, contractAddressHex, args...)
}
