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
	ProviderUrl               string
	ReporterPk                string
	BlockchainType            BlockchainType
	UseAdditionalProviderUrls bool
	UseAdditionalWallets      bool
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

func WithoutAdditionalProviderUrls() ChainHelperOption {
	return func(c *ChainHelperConfig) {
		c.UseAdditionalProviderUrls = false
	}
}

func WithoutAdditionalWallets() ChainHelperOption {
	return func(c *ChainHelperConfig) {
		c.UseAdditionalWallets = false
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
	Kaia BlockchainType = iota
	Ethereum
)

var dialFuncs = map[BlockchainType]func(url string) (utils.ClientInterface, error){
	Kaia: func(rawurl string) (utils.ClientInterface, error) {
		return client.Dial(rawurl)
	},
	Ethereum: func(rawurl string) (utils.ClientInterface, error) {
		return eth_client.Dial(rawurl)
	},
}

const (
	DelegatorEndpoint = "/api/v1/sign/v2"

	EnvDelegatorUrl = "DELEGATOR_URL"
	KaiaProviderUrl = "KAIA_PROVIDER_URL"
	KaiaReporterPk  = "KAIA_REPORTER_PK"
	SignerPk        = "SIGNER_PK"
	EthProviderUrl  = "ETH_PROVIDER_URL"
	EthReporterPk   = "ETH_REPORTER_PK"
)
