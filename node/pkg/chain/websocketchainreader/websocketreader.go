package websocketchainreader

import (
	"context"
	"math/big"
	"time"

	"bisonai.com/miko/node/pkg/chain/eth_client"
	"bisonai.com/miko/node/pkg/chain/utils"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"github.com/kaiachain/kaia"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/client"
	"github.com/kaiachain/kaia/common"
	"github.com/rs/zerolog/log"
)

func New(opts ...ChainReaderOption) (*ChainReader, error) {
	config := &ChainReaderConfig{
		RetryInterval: 1 * time.Second,
	}
	for _, opt := range opts {
		opt(config)
	}

	if config.EthWebsocketUrl == "" || config.KaiaWebsocketUrl == "" || config.BSCWebsocketUrl == "" || config.PolygonWebsocketUrl == "" {
		log.Error().Msg("EthWebsocketUrl, KaiaWebsocketUrl, BSCWebsocketUrl, and PolygonWebsocketUrl must be set")
		return nil, errorSentinel.ErrChainWebsocketUrlNotProvided
	}

	chainIdToChainType := make(map[string]BlockchainType)

	var err error

	var ethClient *eth_client.EthClient
	var ethChainId *big.Int
	if config.EthWebsocketUrl != "" {
		ethClient, err = eth_client.Dial(config.EthWebsocketUrl)
		if err != nil {
			return nil, err
		}

		ethChainId, err = ethClient.ChainID(context.Background())
		if err != nil {
			return nil, err
		}

		chainIdToChainType[ethChainId.String()] = Ethereum
	}

	var kaiaClient *client.Client
	var kaiaChainId *big.Int
	if config.KaiaWebsocketUrl != "" {
		kaiaClient, err = client.Dial(config.KaiaWebsocketUrl)
		if err != nil {
			return nil, err
		}

		kaiaChainId, err = kaiaClient.ChainID(context.Background())
		if err != nil {
			return nil, err
		}

		chainIdToChainType[kaiaChainId.String()] = Kaia
	}

	var (
		bscClient  *eth_client.EthClient
		bscChainId *big.Int
	)
	if config.BSCWebsocketUrl != "" {
		bscClient, err = eth_client.Dial(config.BSCWebsocketUrl)
		if err != nil {
			return nil, err
		}

		bscChainId, err = bscClient.ChainID(context.Background())
		if err != nil {
			return nil, err
		}

		chainIdToChainType[bscChainId.String()] = BSC
	}

	var (
		polygonClient  *eth_client.EthClient
		polygonChainId *big.Int
	)
	if config.PolygonWebsocketUrl != "" {
		polygonClient, err = eth_client.Dial(config.PolygonWebsocketUrl)
		if err != nil {
			return nil, err
		}

		polygonChainId, err = polygonClient.ChainID(context.Background())
		if err != nil {
			return nil, err
		}

		chainIdToChainType[polygonChainId.String()] = Polygon
	}

	// Base is optional — environments that have not yet provisioned a Base RPC
	// (e.g. nodes running before BASE_WEBSOCKET_URL was added to Vault/values)
	// keep working without it instead of failing to start.
	var (
		baseClient  *eth_client.EthClient
		baseChainId *big.Int
	)
	if config.BaseWebsocketUrl != "" {
		baseClient, err = eth_client.Dial(config.BaseWebsocketUrl)
		if err != nil {
			return nil, err
		}

		baseChainId, err = baseClient.ChainID(context.Background())
		if err != nil {
			return nil, err
		}

		chainIdToChainType[baseChainId.String()] = Base
	}

	// Arbitrum is optional for the same reason as Base — environments
	// without ARBITRUM_WEBSOCKET_URL keep working but DEX feeds on
	// Arbitrum will be skipped at chain-type lookup time.
	var (
		arbitrumClient  *eth_client.EthClient
		arbitrumChainId *big.Int
	)
	if config.ArbitrumWebsocketUrl != "" {
		arbitrumClient, err = eth_client.Dial(config.ArbitrumWebsocketUrl)
		if err != nil {
			return nil, err
		}

		arbitrumChainId, err = arbitrumClient.ChainID(context.Background())
		if err != nil {
			return nil, err
		}

		chainIdToChainType[arbitrumChainId.String()] = Arbitrum
	}

	return &ChainReader{
		EthClient:          ethClient,
		KaiaClient:         kaiaClient,
		BscClient:          bscClient,
		PolygonClient:      polygonClient,
		BaseClient:         baseClient,
		ArbitrumClient:     arbitrumClient,
		RetryPeriod:        config.RetryInterval,
		ChainIdToChainType: chainIdToChainType,
	}, nil
}

// Close closes all dialed clients. Unset optional chains (typed-nil) are skipped.
func (c *ChainReader) Close() {
	for _, cl := range []utils.ClientInterface{c.EthClient, c.KaiaClient, c.BscClient, c.PolygonClient, c.BaseClient, c.ArbitrumClient} {
		switch v := cl.(type) {
		case nil:
			continue
		case *eth_client.EthClient:
			if v == nil {
				continue
			}
		case *client.Client:
			if v == nil {
				continue
			}
		}
		cl.Close()
	}
}

func (c *ChainReader) BlockNumber(ctx context.Context, chainType BlockchainType) (*big.Int, error) {
	websocketClient := c.client(chainType)
	return websocketClient.BlockNumber(ctx)
}

func (c *ChainReader) Subscribe(ctx context.Context, opts ...SubscribeOption) error {
	config := &SubscribeConfig{
		ChainType: Ethereum,
	}

	for _, opt := range opts {
		opt(config)
	}

	if config.Address == "" {
		return errorSentinel.ErrChainWebsocketContractAddressNotfound
	}

	if config.Ch == nil {
		return errorSentinel.ErrChainWebsocketChannelNotfound
	}

	go c.handleSubscription(ctx, config)
	return nil
}

func (c *ChainReader) handleSubscription(ctx context.Context, config *SubscribeConfig) {
	// startBlock is resolved once (config.BlockNumber or head at first
	// subscribe) so a resubscribe never jumps to the head at reconnect time.
	var startBlock *big.Int
	cursor := &logCursor{}
	for {
		if startBlock == nil {
			blockNumber, err := c.getBlockNumber(ctx, config)
			if err != nil {
				log.Error().Err(err).Msg("Failed to get block number, retrying")
				if !retryWithContext(ctx, c.RetryPeriod) {
					return
				}
				continue
			}
			startBlock = blockNumber
		}

		query := kaia.FilterQuery{
			FromBlock: cursor.fromBlock(startBlock),
			Addresses: []common.Address{common.HexToAddress(config.Address)},
			Topics:    config.Topics,
		}

		logs := make(chan types.Log)
		sub, err := c.client(config.ChainType).SubscribeFilterLogs(ctx, query, logs)
		if err != nil {
			log.Error().Err(err).Msg("Failed to subscribe, retrying")
			if !retryWithContext(ctx, c.RetryPeriod) {
				return
			}
			continue
		}

		processLogs(ctx, sub, logs, config.Ch, cursor)
		sub.Unsubscribe()

		if !retryWithContext(ctx, c.RetryPeriod) {
			return
		}
		log.Debug().Str("Player", "ChainReader").Msg("Retrying subscription")
	}
}

func (c *ChainReader) getBlockNumber(ctx context.Context, config *SubscribeConfig) (*big.Int, error) {
	if config.BlockNumber != nil {
		return new(big.Int).Set(config.BlockNumber), nil
	}

	return c.client(config.ChainType).BlockNumber(ctx)
}

type logKey struct {
	txHash common.Hash
	index  uint
}

// logCursor tracks the highest block a log was delivered from and which logs
// of that block were already delivered. A resubscribe starts at that block
// (FromBlock is inclusive) rather than block+1, so later logs of the same
// block are not lost; logs already delivered from it are dropped as
// duplicates, since DEX consumers stamp every log with time.Now().
type logCursor struct {
	set   bool
	block uint64
	seen  map[logKey]struct{}
}

func (lc *logCursor) fromBlock(startBlock *big.Int) *big.Int {
	if !lc.set {
		return startBlock
	}
	return new(big.Int).SetUint64(lc.block)
}

// isDuplicate reports whether vLog was already delivered from the cursor block.
func (lc *logCursor) isDuplicate(vLog types.Log) bool {
	if !lc.set || vLog.BlockNumber != lc.block {
		return false
	}
	_, ok := lc.seen[logKey{vLog.TxHash, vLog.Index}]
	return ok
}

func (lc *logCursor) markDelivered(vLog types.Log) {
	if !lc.set || vLog.BlockNumber > lc.block {
		lc.set = true
		lc.block = vLog.BlockNumber
		lc.seen = make(map[logKey]struct{})
	}
	if vLog.BlockNumber == lc.block {
		lc.seen[logKey{vLog.TxHash, vLog.Index}] = struct{}{}
	}
}

func (c *ChainReader) client(chainType BlockchainType) utils.ClientInterface {
	if chainType == Ethereum {
		return c.EthClient
	}

	if chainType == BSC {
		return c.BscClient
	}

	if chainType == Polygon {
		return c.PolygonClient
	}

	if chainType == Base {
		return c.BaseClient
	}

	if chainType == Arbitrum {
		return c.ArbitrumClient
	}

	return c.KaiaClient
}

func (c *ChainReader) ReadContractOnce(ctx context.Context, chain BlockchainType, contractAddressHex string, functionString string, args ...interface{}) (interface{}, error) {
	return utils.ReadContract(ctx, c.client(chain), functionString, contractAddressHex, args...)
}

func retryWithContext(ctx context.Context, duration time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(duration):
		return true
	}
}

// processLogs forwards logs to ch until the subscription fails or ctx is done.
func processLogs(ctx context.Context, sub kaia.Subscription, logs <-chan types.Log, ch chan<- types.Log, cursor *logCursor) {
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-sub.Err():
			log.Warn().Err(err).Msg("Error in subscription")
			return
		case vLog := <-logs:
			if cursor.isDuplicate(vLog) {
				continue
			}
			select {
			case ch <- vLog:
				cursor.markDelivered(vLog)
			case <-ctx.Done():
				return
			}
		}
	}
}
