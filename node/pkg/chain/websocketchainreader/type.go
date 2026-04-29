package websocketchainreader

import (
	"math/big"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/common"
)

type BlockchainType int

const (
	Kaia     BlockchainType = 1
	Ethereum BlockchainType = 2
	BSC      BlockchainType = 3
	Polygon  BlockchainType = 4
	Base     BlockchainType = 5
	Arbitrum BlockchainType = 6
)

type ChainReaderConfig struct {
	KaiaWebsocketUrl     string
	EthWebsocketUrl      string
	BSCWebsocketUrl      string
	PolygonWebsocketUrl  string
	BaseWebsocketUrl     string
	ArbitrumWebsocketUrl string
	RetryInterval        time.Duration
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

func WithBaseWebsocketUrl(url string) ChainReaderOption {
	return func(c *ChainReaderConfig) {
		c.BaseWebsocketUrl = url
	}
}

func WithArbitrumWebsocketUrl(url string) ChainReaderOption {
	return func(c *ChainReaderConfig) {
		c.ArbitrumWebsocketUrl = url
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
	BaseClient         utils.ClientInterface
	ArbitrumClient     utils.ClientInterface
	RetryPeriod        time.Duration
	ChainIdToChainType map[string]BlockchainType
}

type SubscribeConfig struct {
	Address     string
	Ch          chan<- types.Log
	ChainType   BlockchainType
	BlockNumber *big.Int
	// Topics is an optional event-topic filter passed straight through to
	// the chain's eth_subscribe/logs filter.  Empty (nil or zero-length)
	// means "any topic" — same as before this field existed.  The slice
	// follows go-ethereum FilterQuery semantics: each outer element
	// constrains topic[i] (an inner OR of allowed values).  Used by the
	// Uniswap V4 fetcher to subscribe to PoolManager Swap events filtered
	// by the indexed pool id (otherwise we'd receive every swap on the
	// chain, not just our target pools).
	Topics [][]common.Hash
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

// WithTopics sets the optional topic filter for the subscription.  The
// outer slice is positional — element i constrains the i-th topic of the
// log — and the inner slice is an OR of allowed values.  Pass a single
// indexed value (e.g. PoolId) like
//
//	WithTopics([][]common.Hash{{eventSigHash}, {poolIdHash}})
func WithTopics(topics [][]common.Hash) SubscribeOption {
	return func(c *SubscribeConfig) {
		c.Topics = topics
	}
}
