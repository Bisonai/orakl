package websocketchainreader

import (
	"math/big"
	"time"

	"bisonai.com/orakl/node/pkg/chain/utils"
	"github.com/klaytn/klaytn/blockchain/types"
)

const Event = `event Swap(
    address indexed sender,
    address indexed recipient,
    int256 amount0,
    int256 amount1,
    uint160 sqrtPriceX96,
    uint128 liquidity,
    int24 tick
  )`

type BlockchainType int

const (
	Kaia BlockchainType = iota
	Ethereum
)

type ChainReader struct {
	KaiaClient         utils.ClientInterface
	EthClient          utils.ClientInterface
	RetryPeriod        time.Duration
	ChainIdToChainType map[string]BlockchainType
}

type SubscribeConfig struct {
	Address     string
	Ch          chan<- types.Log
	ChainType   BlockchainType
	BlockNumber *big.Int
}

type SubscribeOption func(*SubscribeConfig)

func WithAddress(address string) SubscribeOption {
	return func(c *SubscribeConfig) {
		c.Address = address
	}
}

func WithChannel(ch chan<- types.Log) SubscribeOption {
	return func(c *SubscribeConfig) {
		c.Ch = ch
	}
}

func WithChainType(chainType BlockchainType) SubscribeOption {
	return func(c *SubscribeConfig) {
		c.ChainType = chainType
	}
}

func WithBlockNumber(blockNumber *big.Int) SubscribeOption {
	return func(c *SubscribeConfig) {
		c.BlockNumber = blockNumber
	}
}
