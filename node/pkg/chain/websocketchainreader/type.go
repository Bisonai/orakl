package websocketchainreader

import (
	"math/big"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
	"github.com/kaiachain/kaia/blockchain/types"
)

type BlockchainType int

const (
	Kaia     BlockchainType = 1
	Ethereum BlockchainType = 2
	BSC      BlockchainType = 3
	Polygon  BlockchainType = 4
)

type ChainReaderConfig struct {
	KaiaWebsocketUrl    string
	EthWebsocketUrl     string
	BSCWebsocketUrl     string
	PolygonWebsocketUrl string
	RetryInterval       time.Duration
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

func WithBSCWebsocketUrl(url string) ChainReaderOption {
	return func(c *ChainReaderConfig) {
		c.BSCWebsocketUrl = url
	}
}

func WithPolygonWebsocketUrl(url string) ChainReaderOption {
	return func(c *ChainReaderConfig) {
		c.PolygonWebsocketUrl = url
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
	BscClient          utils.ClientInterface
	PolygonClient      utils.ClientInterface
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
