package uniswapv4

import (
	"bisonai.com/miko/node/pkg/chain/websocketchainreader"
)

// ChainConfig holds the per-chain Uniswap V4 contract addresses.  All V4
// pools on a given chain live inside the singleton PoolManager and are
// read through the StateView helper.  Source:
// https://docs.uniswap.org/contracts/v4/deployments
type ChainConfig struct {
	StateView   string
	PoolManager string
}

// chainConfigs maps the chain type used by the websocketchainreader to
// the relevant V4 addresses.  Add new chains here as Uniswap deploys.
var chainConfigs = map[websocketchainreader.BlockchainType]ChainConfig{
	websocketchainreader.Ethereum: {
		StateView:   "0x7ffe42c4a5deea5b0fec41c94c136cf115597227",
		PoolManager: "0x000000000004444c5dc75cb358380d2e3de08a90",
	},
	websocketchainreader.Polygon: {
		StateView:   "0x5ea1bd7974c8a611cbab0bdcafcb1d9cc9b3ba5a",
		PoolManager: "0x67366782805870060151383f4bbff9dab53e5cd6",
	},
	websocketchainreader.Arbitrum: {
		StateView:   "0x76fd297e2d437cd7f76d50f01afe6160f86e9990",
		PoolManager: "0x360e68faccca8ca495c1b759fd9eee466db9fb32",
	},
	websocketchainreader.Base: {
		StateView:   "0xa3c0c9b65bad0b08107aa264b0f3db444b867a71",
		PoolManager: "0x498581ff718922c3f8e6a244956af099b2652b2b",
	},
}

// LookupChainConfig returns the V4 addresses for chainType, if known.
func LookupChainConfig(chainType websocketchainreader.BlockchainType) (ChainConfig, bool) {
	cfg, ok := chainConfigs[chainType]
	return cfg, ok
}
