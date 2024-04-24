package helper

import (
	"crypto/ecdsa"
	"math/big"

	"bisonai.com/orakl/node/pkg/chain/eth_client"
	"bisonai.com/orakl/node/pkg/chain/utils"
	"github.com/klaytn/klaytn/client"
)

type ChainHelper struct {
	clients             []utils.ClientInterface
	wallets             []string
	chainID             *big.Int
	delegatorUrl        string
	lastUsedWalletIndex int
}

type ChainHelperConfig struct {
	ProviderUrl    string
	ReporterPk     string
	BlockchainType BlockchainType
}

type ChainHelperOption func(*ChainHelperConfig)

func WithProviderUrl(url string) ChainHelperOption {
	return func(c *ChainHelperConfig) {
		c.ProviderUrl = url
	}
}

func WithReporterPk(pk string) ChainHelperOption {
	return func(c *ChainHelperConfig) {
		c.ReporterPk = pk
	}
}

func WithBlockchainType(t BlockchainType) ChainHelperOption {
	return func(c *ChainHelperConfig) {
		c.BlockchainType = t
	}
}

type SignHelper struct {
	PK *ecdsa.PrivateKey
}

type signedTx struct {
	SignedRawTx *string `json:"signedRawTx" db:"signedRawTx"`
}

type BlockchainType int

const (
	Klaytn BlockchainType = iota
	Ethereum
)

var dialFuncs = map[BlockchainType]func(url string) (utils.ClientInterface, error){
	Klaytn: func(rawurl string) (utils.ClientInterface, error) {
		return client.Dial(rawurl)
	},
	Ethereum: func(rawurl string) (utils.ClientInterface, error) {
		return eth_client.Dial(rawurl)
	},
}

const (
	DelegatorEndpoint = "/api/v1/sign/volatile"

	EnvDelegatorUrl   = "DELEGATOR_URL"
	KlaytnProviderUrl = "KLAYTN_PROVIDER_URL"
	KlaytnReporterPk  = "KLAYTN_REPORTER_PK"
	EthProviderUrl    = "ETH_PROVIDER_URL"
	EthReporterPk     = "ETH_REPORTER_PK"
)