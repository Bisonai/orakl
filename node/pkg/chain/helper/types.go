package helper

import (
	"crypto/ecdsa"
	"math/big"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/chain/eth_client"
	"bisonai.com/miko/node/pkg/chain/noncemanagerv2"
	"bisonai.com/miko/node/pkg/chain/utils"
	"github.com/kaiachain/kaia/client"
	"github.com/kaiachain/kaia/common"
)

type ChainHelper struct {
	client       utils.ClientInterface
	wallet       string
	chainID      *big.Int
	delegatorUrl string
	noncemanager *noncemanagerv2.NonceManagerV2
}

type ChainHelperConfig struct {
	ProviderUrl               string
	ReporterPk                string
	BlockchainType            BlockchainType
	UseAdditionalProviderUrls bool
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

// Signer owns the node's global-aggregate signing key and keeps it reconciled with the
// on-chain SubmissionProxy oracle whitelist. The on-chain whitelist — never local state — is
// the authority on which key may sign; see reconcile/rotate in signer.go (issue #2516).
type Signer struct {
	PK    *ecdsa.PrivateKey
	chain oracleChain // on-chain oracle reads/writes (injectable for tests)
	store signerStore // durable keyring (injectable for tests)

	// mu guards the fast sign-path fields below.
	mu               sync.RWMutex
	activeAddr       common.Address // address of PK
	cachedExpiration time.Time      // on-chain expirationTime of activeAddr (authoritative sign-gate input)
	lastConfirmedAt  time.Time      // last time activeAddr was positively confirmed whitelisted on-chain
	usable           bool           // false => refuse to sign (fail loud) instead of signing with a stale key
	rotating         bool           // true while a rotation is in flight => refuse to sign

	rotateMu sync.Mutex // serializes reconcile+rotate within this process (TryLock)

	staticMode         bool // WithSignerPk: fixed key, no reconcile/rotation/confirmation gate
	bootstrapPk        string
	renewThreshold     time.Duration
	livenessInterval   time.Duration
	skewMargin         time.Duration
	confirmationTTL    time.Duration // refuse to sign if activeAddr has not been confirmed whitelisted within this window
	verifyPollInterval time.Duration // rotation on-chain confirmation poll cadence
	verifyPollMax      int           // rotation on-chain confirmation poll attempts
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

	DelegatorTimeout            = 10 * time.Second
	DefaultSignerRenewInterval  = 12 * time.Hour
	DefaultSignerRenewThreshold = 7 * 24 * time.Hour
	SignerDetailFuncSignature   = "whitelist(address) returns ((uint256, uint256))"
	UpdateSignerFuncSignature   = "updateOracle(address) returns (uint256)"
	GetAllOraclesFuncSignature  = "getAllOracles() public view returns (address[] memory)"

	// Crash-safe rotation / reconciliation tunables (issue #2516).
	DefaultSignerLivenessInterval = 30 * time.Second // how often the active key's on-chain status is re-checked
	DefaultSignerSkewMargin       = 90 * time.Second // refuse to sign this long before the cached expiration
	signerVerifyPollInterval      = 3 * time.Second  // poll cadence when confirming a rotation on-chain
	signerVerifyPollMax           = 20               // ~60s budget to observe the updateOracle result
)
