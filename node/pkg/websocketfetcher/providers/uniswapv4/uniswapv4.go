package uniswapv4

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
	"bisonai.com/miko/node/pkg/chain/websocketchainreader"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/kaiachain/kaia/blockchain/types"
	kaiacommon "github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
	"github.com/rs/zerolog/log"
)

// V4Fetcher is the Uniswap V4 implementation.  Unlike V3, every pool on a
// given chain shares one PoolManager singleton — so pools are addressed by
// a 32-byte poolId rather than a deployed contract address.  Reads go
// through the read-only StateView helper (getSlot0(poolId)); live updates
// come from PoolManager Swap events filtered by indexed poolId.
type V4Fetcher common.DexFetcher

const (
	// GET_SLOT0 is the V4 StateView read interface.  Same shape as the
	// V3 IUniswapV3Pool.slot0() output we already consume — just sourced
	// through the helper contract instead of from the pool itself.
	GET_SLOT0 = "function getSlot0(bytes32 poolId) external view returns (uint160 sqrtPriceX96, int24 tick, uint24 protocolFee, uint24 lpFee)"

	// SWAP_EVENT is the PoolManager Swap event.  PoolId is bytes32 in the
	// underlying ABI (the Solidity `type PoolId is bytes32` is purely a
	// compiler nicety) so we name the topic "bytes32 indexed id" directly.
	SWAP_EVENT = `event Swap(
        bytes32 indexed id,
        address indexed sender,
        int128 amount0,
        int128 amount1,
        uint160 sqrtPriceX96,
        uint128 liquidity,
        int24 tick,
        uint24 fee
    )`

	// SwapEventCanonicalSig is the canonical (no-whitespace, no-names)
	// form of SWAP_EVENT used to compute topic[0].  Kept hard-coded so
	// the hash is stable independent of the human-readable signature.
	SwapEventCanonicalSig = "Swap(bytes32,address,int128,int128,uint160,uint128,int24,uint24)"
)

// New constructs a V4 fetcher from the standard DexFetcher options.
func New(opts ...common.DexFetcherOption) common.FetcherInterface {
	config := &common.DexFetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	return &V4Fetcher{
		Feeds:                config.Feeds,
		FeedDataBuffer:       config.FeedDataBuffer,
		WebsocketChainReader: config.WebsocketChainReader,
		LatestEntries:        make(map[int32]*common.FeedData),
	}
}

// Run launches one goroutine per feed, mirroring the V3 providers.  Each
// run() does a one-shot StateView read for the initial price, kicks off a
// best-effort Swap subscription in the background, then enters the
// HeartbeatPoll loop in the foreground.
func (f *V4Fetcher) Run(ctx context.Context) {
	for _, feed := range f.Feeds {
		go f.run(ctx, feed)
		// Same RPC-rate-limit-avoidance sleep as V3 providers.
		time.Sleep(1 * time.Second)
	}
}

func (f *V4Fetcher) run(ctx context.Context, feed common.Feed) {
	// 1. Initial read through StateView.getSlot0(poolId).
	price, err := f.getInitialPrice(ctx, feed)
	if err != nil {
		log.Error().Str("Player", "UniswapV4").Err(err).
			Int32("feedID", feed.ID).Str("name", feed.Name).
			Msg("error in uniswapv4.run, failed to get initial price")
		return
	}

	now := time.Now()
	initialFeedData := &common.FeedData{
		FeedID:    feed.ID,
		Value:     *price,
		Timestamp: &now,
	}
	log.Debug().Str("Player", "UniswapV4").Any("feedData", initialFeedData).Msg("initial price fetched")
	f.FeedDataBuffer <- initialFeedData

	f.Mutex.Lock()
	f.LatestEntries[feed.ID] = initialFeedData
	f.Mutex.Unlock()

	// 2. Best-effort PoolManager Swap subscription (background).  If the
	// subscription fails or returns, the foreground heartbeat poll keeps
	// supplying fresh data — same lifecycle pattern as V3 providers.
	go func() {
		if err := f.subscribeEvent(ctx, feed); err != nil {
			log.Error().Str("Player", "UniswapV4").Err(err).
				Int32("feedID", feed.ID).Str("name", feed.Name).
				Msg("error in uniswapv4.run, subscribe event ended")
		}
	}()

	// 3. Heartbeat poll in the foreground.
	common.HeartbeatPoll(ctx, common.GetDexPollInterval(), "UniswapV4", feed.ID, feed.Name,
		f.getInitialPriceFor(feed),
		func(fd *common.FeedData) {
			f.FeedDataBuffer <- fd
			f.Mutex.Lock()
			f.LatestEntries[feed.ID] = fd
			f.Mutex.Unlock()
		},
	)
}

// getInitialPriceFor returns a function suitable for HeartbeatPoll that
// invokes f.getInitialPrice with the feed bound in.
func (f *V4Fetcher) getInitialPriceFor(feed common.Feed) func(context.Context) (*float64, error) {
	return func(ctx context.Context) (*float64, error) {
		return f.getInitialPrice(ctx, feed)
	}
}

func (f *V4Fetcher) getInitialPrice(ctx context.Context, feed common.Feed) (*float64, error) {
	definition := new(common.DexFeedDefinition)
	if err := json.Unmarshal(feed.Definition, &definition); err != nil {
		log.Error().Str("Player", "UniswapV4").Err(err).Msg("error in uniswapv4.getInitialPrice, failed to unmarshal definition")
		return nil, err
	}
	return f.getPriceThroughSlot0(ctx, definition)
}

// getPriceThroughSlot0 calls StateView.getSlot0(poolId) and converts the
// returned sqrtPriceX96 to a human-readable price.
func (f *V4Fetcher) getPriceThroughSlot0(ctx context.Context, definition *common.DexFeedDefinition) (*float64, error) {
	chainType, ok := f.WebsocketChainReader.ChainIdToChainType[definition.ChainId]
	if !ok {
		log.Error().Str("Player", "UniswapV4").Str("chainId", definition.ChainId).Msg("error in uniswapv4.getPriceThroughSlot0, chain type not found")
		return nil, errorSentinel.ErrFetcherNoMatchingChainID
	}

	chainCfg, ok := LookupChainConfig(chainType)
	if !ok {
		log.Error().Str("Player", "UniswapV4").Str("chainId", definition.ChainId).Msg("error in uniswapv4.getPriceThroughSlot0, no V4 deployment for chain")
		return nil, errorSentinel.ErrFetcherNoMatchingChainID
	}

	poolID, err := ParsePoolID(definition.PoolID)
	if err != nil {
		return nil, err
	}

	rawResult, err := f.WebsocketChainReader.ReadContractOnce(ctx, chainType, chainCfg.StateView, GET_SLOT0, poolID)
	if err != nil {
		log.Error().Str("Player", "UniswapV4").Err(err).Msg("error in uniswapv4.getPriceThroughSlot0, failed to read contract")
		return nil, err
	}

	sqrtPrice, err := extractSqrtPriceX96(rawResult)
	if err != nil {
		return nil, err
	}

	return GetTokenPrice(sqrtPrice, definition)
}

// subscribeEvent subscribes to PoolManager Swap events filtered by the
// feed's indexed poolId.  Without the topic filter we'd receive every
// swap on the chain (PoolManager handles all V4 pools), which is
// untenable on busy chains.
func (f *V4Fetcher) subscribeEvent(ctx context.Context, feed common.Feed) error {
	definition := new(common.DexFeedDefinition)
	if err := json.Unmarshal(feed.Definition, &definition); err != nil {
		log.Error().Str("Player", "UniswapV4").Err(err).Msg("error in uniswapv4.subscribeEvent, failed to unmarshal definition")
		return err
	}

	chainType, ok := f.WebsocketChainReader.ChainIdToChainType[definition.ChainId]
	if !ok {
		log.Error().Str("Player", "UniswapV4").Str("chainId", definition.ChainId).Msg("error in uniswapv4.subscribeEvent, chain type not found")
		return errorSentinel.ErrFetcherNoMatchingChainID
	}

	chainCfg, ok := LookupChainConfig(chainType)
	if !ok {
		return errorSentinel.ErrFetcherNoMatchingChainID
	}

	poolIDBytes, err := ParsePoolID(definition.PoolID)
	if err != nil {
		return err
	}
	poolIDHash := kaiacommon.BytesToHash(poolIDBytes[:])

	eventName, input, _, err := utils.ParseMethodSignature(SWAP_EVENT)
	if err != nil {
		log.Error().Str("Player", "UniswapV4").Err(err).Msg("error in uniswapv4.subscribeEvent, failed to parse method signature")
		return err
	}

	swapEventABI, err := utils.GenerateEventABI(eventName, input)
	if err != nil {
		log.Error().Str("Player", "UniswapV4").Err(err).Msg("error in uniswapv4.subscribeEvent, failed to generate event abi")
		return err
	}

	// topic[0] is keccak256 of the canonical (whitespace-stripped) event
	// signature.  Hard-code that string rather than reusing the
	// human-readable SWAP_EVENT to make the hash stable across cosmetic
	// edits.
	eventSigHash := SwapEventTopic0()

	logChannel := make(chan types.Log)
	if err := f.WebsocketChainReader.Subscribe(
		ctx,
		websocketchainreader.WithAddress(chainCfg.PoolManager),
		websocketchainreader.WithChannel(logChannel),
		websocketchainreader.WithChainType(chainType),
		websocketchainreader.WithTopics([][]kaiacommon.Hash{
			{eventSigHash},
			{poolIDHash},
		}),
	); err != nil {
		log.Error().Str("Player", "UniswapV4").Err(err).Msg("error in uniswapv4.subscribeEvent, failed to subscribe")
		return err
	}

	// Loop driven by select so we exit promptly on ctx cancellation.
	// The websocketchainreader.Subscribe contract does not close
	// logChannel on shutdown, so a `range` over it would leak this
	// goroutine for the lifetime of the process even after ctx is
	// cancelled.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case eventLog, ok := <-logChannel:
			if !ok {
				return nil
			}
			// Defensive: the subscription is already filtered by both
			// the event signature and the pool id, but verify topic[1]
			// matches in case a misbehaving relay forwards extra logs.
			if len(eventLog.Topics) < 2 || eventLog.Topics[1] != poolIDHash {
				continue
			}

			res, err := swapEventABI.Unpack(eventName, eventLog.Data)
			if err != nil {
				continue
			}
			// Non-indexed args, in declaration order:
			// [amount0, amount1, sqrtPriceX96, liquidity, tick, fee]
			if len(res) < 3 {
				continue
			}
			sqrtPrice, ok := res[2].(*big.Int)
			if !ok {
				continue
			}
			price, err := GetTokenPrice(sqrtPrice, definition)
			if err != nil {
				log.Error().Str("Player", "UniswapV4").Err(err).Msg("error in uniswapv4.subscribeEvent, failed to get token price")
				continue
			}
			now := time.Now()
			feedData := &common.FeedData{
				FeedID:    feed.ID,
				Value:     *price,
				Timestamp: &now,
			}
			log.Debug().Str("Player", "UniswapV4").Any("feedData", feedData).Msg("price fetched")
			f.FeedDataBuffer <- feedData
			f.Mutex.Lock()
			f.LatestEntries[feed.ID] = feedData
			f.Mutex.Unlock()
		}
	}
}

// SwapEventTopic0 returns keccak256 of SwapEventCanonicalSig.  This is
// what the chain places in topic[0] of every PoolManager Swap log.
func SwapEventTopic0() kaiacommon.Hash {
	return kaiacommon.BytesToHash(crypto.Keccak256([]byte(SwapEventCanonicalSig)))
}

// ParsePoolID accepts a hex-encoded 32-byte pool identifier (with or
// without the "0x" prefix, case-insensitive) and returns it as a
// fixed-size [32]byte suitable for ABI bytes32 packing.
func ParsePoolID(s string) ([32]byte, error) {
	var out [32]byte
	if s == "" {
		return out, errors.New("uniswapv4: empty poolId")
	}
	clean := strings.TrimPrefix(strings.ToLower(s), "0x")
	if len(clean) != 64 {
		return out, errors.New("uniswapv4: poolId must be 32 bytes (64 hex chars)")
	}
	raw, err := hex.DecodeString(clean)
	if err != nil {
		return out, err
	}
	copy(out[:], raw)
	return out, nil
}

// extractSqrtPriceX96 pulls the first element of an ABI getSlot0 tuple
// result and casts it to *big.Int.  Returns a typed sentinel error so
// callers can distinguish "result shape was wrong" from generic failure.
func extractSqrtPriceX96(rawResult interface{}) (*big.Int, error) {
	rawResultSlice, ok := rawResult.([]interface{})
	if !ok || len(rawResultSlice) < 1 {
		return nil, errorSentinel.ErrFetcherFailedToGetDexResultSlice
	}
	sqrtPrice, ok := rawResultSlice[0].(*big.Int)
	if !ok {
		return nil, errorSentinel.ErrFetcherFailedBigIntConvert
	}
	return sqrtPrice, nil
}

// GetTokenPrice converts a Uniswap-style sqrtPriceX96 into the
// human-readable price of token0 expressed in token1 (or its reciprocal,
// when definition.Reciprocal is true).
//
// Math is identical to the V3 providers — this lives here purely to keep
// the V4 package self-contained.  See providers/uniswap/uniswap.go and
// providers/pancakeswap/pancakeswap.go for the matching implementations.
func GetTokenPrice(sqrtPrice *big.Int, definition *common.DexFeedDefinition) (*float64, error) {
	if sqrtPrice == nil {
		return nil, errorSentinel.ErrFetcherInvalidInput
	}

	decimal0 := definition.Token0Decimals
	decimal1 := definition.Token1Decimals

	sqrtPriceX96Float := new(big.Float).SetInt(sqrtPrice)
	sqrtPriceX96Float.Quo(sqrtPriceX96Float, new(big.Float).SetFloat64(math.Pow(2, 96)))
	sqrtPriceX96Float.Mul(sqrtPriceX96Float, sqrtPriceX96Float)

	decimalDiff := new(big.Float).SetFloat64(math.Pow(10, float64(decimal1-decimal0)))
	datum := sqrtPriceX96Float.Quo(sqrtPriceX96Float, decimalDiff)

	if definition.Reciprocal != nil && *definition.Reciprocal {
		if datum == nil || datum.Sign() == 0 {
			return nil, errorSentinel.ErrFetcherDivisionByZero
		}
		datum = datum.Quo(new(big.Float).SetFloat64(1), datum)
	}

	result, _ := datum.Float64()
	return &result, nil
}
