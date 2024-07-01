package websocketchainreader

import (
	"math/big"
	"time"

	"bisonai.com/orakl/node/pkg/chain/utils"
	"github.com/klaytn/klaytn/blockchain/types"
)

type BlockchainType int

const (
	Kaia     BlockchainType = 1
	Ethereum BlockchainType = 2
)

type ChainReaderConfig struct {
	KaiaWebsocketUrl string
	EthWebsocketUrl  string
	RetryInterval    time.Duration
}

type ChainReaderOption func(*ChainReaderConfig)

func WithKaiaWebsocketUrl(url string) ChainReaderOption {
	return func(c *ChainReaderConfig) {
		c.KaiaWebsocketUrl = url
	}
}

func WithEthWebsocketUrl(url string) ChainReaderOption {
	return func(c *ChainReaderConfig) {
		c.EthWebsocketUrl = url
	}
}

func WithRetryInterval(interval time.Duration) ChainReaderOption {
	return func(c *ChainReaderConfig) {
		c.RetryInterval = interval
	}
}

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

func WithStartBlockNumber(blockNumber *big.Int) SubscribeOption {
	return func(c *SubscribeConfig) {
		c.BlockNumber = blockNumber
	}
}
