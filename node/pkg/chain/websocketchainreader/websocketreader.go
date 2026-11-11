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

// Log subscriptions only stream newly mined logs (FromBlock filters, it does
// not replay history), so logs emitted while disconnected are backfilled with
// eth_getLogs after every resubscribe.
const (
	backfillChunkBlocks = 1000
	// maxBackfillBlocks bounds the backfill after a long outage; older logs
	// are skipped since DEX consumers only care about recent prices.
	maxBackfillBlocks = 10000
	// maxBackfillAttempts consecutive failed backfills resubscribe and retry;
	// after that streaming continues and the gap is accepted.
	maxBackfillAttempts = 3
	// backfillSafetyBlocks: after a backfill the cursor advances only to
	// head-backfillSafetyBlocks, so a getLogs backend lagging behind the head
	// can't hide recent blocks, while a quiet pool doesn't rescan old ranges.
	backfillSafetyBlocks = 32
)

// logFilterer is implemented by both eth_client.EthClient and kaia client.Client.
type logFilterer interface {
	FilterLogs(ctx context.Context, q kaia.FilterQuery) ([]types.Log, error)
}

func (c *ChainReader) handleSubscription(ctx context.Context, config *SubscribeConfig) {
	// cursor is created once (config.BlockNumber or head+1 at first subscribe)
	// so a resubscribe never restarts from the original start or jumps to the
	// head at reconnect time.
	var cursor *logCursor
	backfillFailures := 0
	for {
		if cursor == nil {
			start, err := c.startBlock(ctx, config)
			if err != nil {
				log.Error().Err(err).Msg("Failed to get block number, retrying")
				if !retryWithContext(ctx, c.RetryPeriod) {
					return
				}
				log.Debug().Str("Player", "ChainReader").Msg("Retrying subscription")
				continue
			}
			cursor = &logCursor{block: start}
		}

		query := kaia.FilterQuery{
			FromBlock: new(big.Int).SetUint64(cursor.block),
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
			log.Debug().Str("Player", "ChainReader").Msg("Retrying subscription")
			continue
		}

		// Subscribe first, then backfill, so no block falls between the two
		// (this also covers blocks mined between startBlock and subscribe).
		// Logs streamed meanwhile queue in the RPC client's subscription
		// buffer and are deduped by the cursor.
		stream := true
		if err := c.backfill(ctx, config, query, cursor); err != nil {
			backfillFailures++
			if backfillFailures < maxBackfillAttempts {
				log.Warn().Err(err).Str("Player", "ChainReader").Int("attempt", backfillFailures).Msg("Backfill failed, resubscribing")
				stream = false
			} else {
				log.Warn().Err(err).Str("Player", "ChainReader").Msg("Backfill failed, streaming without it, logs may be missed")
			}
		}
		if stream {
			backfillFailures = 0
			processLogs(ctx, sub, logs, config.Ch, cursor)
		}
		sub.Unsubscribe()

		if !retryWithContext(ctx, c.RetryPeriod) {
			return
		}
		log.Debug().Str("Player", "ChainReader").Msg("Retrying subscription")
	}
}

func (c *ChainReader) startBlock(ctx context.Context, config *SubscribeConfig) (uint64, error) {
	if config.BlockNumber != nil {
		return config.BlockNumber.Uint64(), nil
	}

	head, err := c.client(config.ChainType).BlockNumber(ctx)
	if err != nil {
		return 0, err
	}
	// Only logs mined after the first subscribe, as before.
	return head.Uint64() + 1, nil
}

// backfill delivers logs from cursor.block up to the current head.
func (c *ChainReader) backfill(ctx context.Context, config *SubscribeConfig, query kaia.FilterQuery, cursor *logCursor) error {
	cl := c.client(config.ChainType)
	filterer, ok := cl.(logFilterer)
	if !ok {
		log.Warn().Str("Player", "ChainReader").Msg("Client has no FilterLogs, skipping backfill, logs may be missed")
		return nil
	}

	headBig, err := cl.BlockNumber(ctx)
	if err != nil {
		return err
	}
	head := headBig.Uint64()
	from := cursor.block
	if from > head {
		return nil
	}
	if head-from+1 > maxBackfillBlocks {
		log.Warn().Str("Player", "ChainReader").Uint64("from", from).Uint64("head", head).Msg("Backfill gap too large, skipping older blocks")
		from = head - maxBackfillBlocks + 1
	}

	for chunkStart := from; chunkStart <= head; chunkStart += backfillChunkBlocks {
		chunkEnd := min(chunkStart+backfillChunkBlocks-1, head)
		q := query
		q.FromBlock = new(big.Int).SetUint64(chunkStart)
		q.ToBlock = new(big.Int).SetUint64(chunkEnd)
		got, err := filterer.FilterLogs(ctx, q)
		if err != nil {
			return err
		}
		for _, vLog := range got {
			if !deliverLog(ctx, config.Ch, cursor, vLog) {
				return ctx.Err()
			}
		}
	}
	if head >= backfillSafetyBlocks {
		cursor.advanceTo(head - backfillSafetyBlocks + 1)
	}
	return nil
}

// logCursor is the position of the next log to deliver. Log Index is unique
// and increasing within a block, so (block, index) orders logs. A resubscribe
// uses FromBlock = cursor.block (inclusive, not block+1) so later logs of a
// partially delivered block are not lost; logs before the cursor are dropped
// as duplicates, since DEX consumers stamp every log with time.Now().
type logCursor struct {
	block     uint64
	index     uint
	blockHash common.Hash
}

// isNew reports whether vLog is past the cursor, or belongs to a different
// (reorged-in) version of the cursor block. Deeper reorgs are not tracked.
func (lc *logCursor) isNew(vLog types.Log) bool {
	if vLog.BlockNumber == lc.block && lc.index > 0 && vLog.BlockHash != lc.blockHash {
		return true
	}
	return vLog.BlockNumber > lc.block || (vLog.BlockNumber == lc.block && vLog.Index >= lc.index)
}

func (lc *logCursor) markDelivered(vLog types.Log) {
	lc.block = vLog.BlockNumber
	lc.index = vLog.Index + 1
	lc.blockHash = vLog.BlockHash
}

func (lc *logCursor) advanceTo(block uint64) {
	if block > lc.block {
		lc.block = block
		lc.index = 0
		lc.blockHash = common.Hash{}
	}
}

// deliverLog sends vLog to ch unless it was already delivered or was removed
// by a reorg (consumers would treat an orphaned swap as a fresh price).
// Returns false when ctx is done.
func deliverLog(ctx context.Context, ch chan<- types.Log, cursor *logCursor, vLog types.Log) bool {
	if vLog.Removed || !cursor.isNew(vLog) {
		return true
	}
	select {
	case ch <- vLog:
		cursor.markDelivered(vLog)
		return true
	case <-ctx.Done():
		return false
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
			if !deliverLog(ctx, ch, cursor, vLog) {
				return
			}
		}
	}
}
