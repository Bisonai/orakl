package uniswap

import (
	"context"
	"encoding/json"
	"math"
	"math/big"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
	"bisonai.com/miko/node/pkg/chain/websocketchainreader"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/rs/zerolog/log"
)

type UniswapFetcher common.DexFetcher

const (
	SLOT0 = "function slot0() external view returns (uint160 sqrtPriceX96, int24 tick, uint16 observationIndex, uint16 observationCardinality, uint16 observationCardinalityNext, uint8 feeProtocol, bool unlocked)"
	EVENT = `event Swap(
    address indexed sender,
    address indexed recipient,
    int256 amount0,
    int256 amount1,
    uint160 sqrtPriceX96,
    uint128 liquidity,
    int24 tick
  )`
)

func New(opts ...common.DexFetcherOption) common.FetcherInterface {
	config := &common.DexFetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	return &UniswapFetcher{
		Feeds:                config.Feeds,
		FeedDataBuffer:       config.FeedDataBuffer,
		WebsocketChainReader: config.WebsocketChainReader,
		LatestEntries:        make(map[int32]*common.FeedData),
	}
}

func (f *UniswapFetcher) Run(ctx context.Context) {
	for _, feed := range f.Feeds {
		go f.run(ctx, feed)
		// sleep to avoid blockage from json rpc url rate limitation
		time.Sleep(1 * time.Second)
	}
}

func (f *UniswapFetcher) run(ctx context.Context, feed common.Feed) {
	// 1. get initial data once through contract call
	price, err := f.getInitialPrice(ctx, feed)
	if err != nil {
		log.Error().Str("Player", "Uniswap").Err(err).Msg("error in uniswap.run, failed to get initial price")
		return
	}

	now := time.Now()
	initialFeedData := &common.FeedData{
		FeedID:    feed.ID,
		Value:     *price,
		Timestamp: &now,
	}
	log.Debug().Str("Player", "Uniswap").Any("feedData", initialFeedData).Msg("initial price fetched")
	f.FeedDataBuffer <- initialFeedData

	f.Mutex.Lock()
	f.LatestEntries[feed.ID] = initialFeedData
	f.Mutex.Unlock()

	// 2. regular fetch if data is older than 1hr
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	go func() {
		for {
			select {
			case <-t.C:
				f.Mutex.Lock()
				last, ok := f.LatestEntries[feed.ID]
				f.Mutex.Unlock()
				if !ok {
					continue
				}

				now := time.Now()
				if last.Timestamp.Before(now.Add(-time.Hour)) {
					price, err := f.getInitialPrice(ctx, feed)
					if err != nil {
						log.Error().Str("Player", "Uniswap").Err(err).Msg("failed to get refreshed price")
						continue
					}

					refreshedFeedData := &common.FeedData{
						FeedID:    feed.ID,
						Value:     *price,
						Timestamp: &now,
					}
					f.FeedDataBuffer <- refreshedFeedData

					f.Mutex.Lock()
					f.LatestEntries[feed.ID] = refreshedFeedData
					f.Mutex.Unlock()
				}
			case <-ctx.Done():
				return
			}
		}

	}()

	// 3. get subsequent data through websocket
	err = f.subscribeEvent(ctx, feed)
	if err != nil {
		log.Error().Str("Player", "Uniswap").Err(err).Msg("error in uniswap.run, failed to subscribe event")
		return
	}
}

func (f *UniswapFetcher) getInitialPrice(ctx context.Context, feed common.Feed) (*float64, error) {
	definition := new(common.DexFeedDefinition)
	err := json.Unmarshal(feed.Definition, &definition)
	if err != nil {
		log.Error().Str("Player", "Uniswap").Err(err).Msg("error in uniswap.getInitialPrice, failed to unmarshal definition")
		return nil, err
	}
	return f.getPriceThroughSlotCall(ctx, definition)
}

func (f *UniswapFetcher) subscribeEvent(ctx context.Context, feed common.Feed) error {
	definition := new(common.DexFeedDefinition)
	err := json.Unmarshal(feed.Definition, &definition)
	if err != nil {
		log.Error().Str("Player", "Uniswap").Err(err).Msg("error in uniswap.getInitialPrice, failed to unmarshal definition")
		return err
	}

	return f.readSwapEvent(ctx, feed, definition)
}

func (f *UniswapFetcher) getPriceThroughSlotCall(ctx context.Context, definition *common.DexFeedDefinition) (*float64, error) {
	chainType, ok := f.WebsocketChainReader.ChainIdToChainType[definition.ChainId]
	if !ok {
		log.Error().Str("Player", "Uniswap").Str("chainId", definition.ChainId).Msg("error in uniswap.getInitialPrice, chain type not found")
		return nil, errorSentinel.ErrFetcherNoMatchingChainID
	}

	rawResult, err := f.WebsocketChainReader.ReadContractOnce(ctx, chainType, definition.Address, SLOT0)
	if err != nil {
		log.Error().Str("Player", "Uniswap").Err(err).Msg("error in uniswap.getInitialPrice, failed to read contract")
		return nil, err
	}

	rawResultSlice, ok := rawResult.([]interface{})
	if !ok || len(rawResultSlice) < 1 {
		log.Error().Str("Player", "Uniswap").Msg("error in uniswap.getInitialPrice, failed to get slice result")
		return nil, errorSentinel.ErrFetcherFailedToGetDexResultSlice
	}

	sqrtPrice, ok := rawResultSlice[0].(*big.Int)
	if !ok {
		log.Error().Str("Player", "Uniswap").Msg("error in uniswap.getInitialPrice, failed to convert raw price")
		return nil, errorSentinel.ErrFetcherFailedBigIntConvert
	}

	return getTokenPrice(sqrtPrice, definition)
}

func (f *UniswapFetcher) readSwapEvent(ctx context.Context, feed common.Feed, definition *common.DexFeedDefinition) error {
	logChannel := make(chan types.Log)
	address := definition.Address

	chainType, ok := f.WebsocketChainReader.ChainIdToChainType[definition.ChainId]
	if !ok {
		log.Error().Str("Player", "Uniswap").Str("chainId", definition.ChainId).Msg("error in uniswap.getInitialPrice, chain type not found")
		return errorSentinel.ErrFetcherNoMatchingChainID
	}

	var eventName, input string
	var eventParseErr error

	eventName, input, _, eventParseErr = utils.ParseMethodSignature(EVENT)
	if eventParseErr != nil {
		log.Error().Str("Player", "Uniswap").Err(eventParseErr).Msg("error in uniswap.subscribeEvent, failed to parse method signature")
		return eventParseErr
	}

	swapEventABI, err := utils.GenerateEventABI(eventName, input)
	if err != nil {
		log.Error().Str("Player", "Uniswap").Err(err).Msg("error in uniswap.subscribeEvent, failed to generate event abi")
		return err
	}

	err = f.WebsocketChainReader.Subscribe(
		ctx,
		websocketchainreader.WithAddress(address),
		websocketchainreader.WithChannel(logChannel),
		websocketchainreader.WithChainType(chainType))
	if err != nil {
		log.Error().Str("Player", "Uniswap").Err(err).Msg("error in uniswap.subscribeEvent, failed to subscribe")
		return err
	}

	for eventLog := range logChannel {
		res, err := swapEventABI.Unpack(eventName, eventLog.Data)
		if err != nil {
			continue
		}
		sqrtPrice := res[2]
		price, err := getTokenPrice(sqrtPrice.(*big.Int), definition)
		if err != nil {
			log.Error().Str("Player", "Uniswap").Err(err).Msg("error in uniswap.subscribeEvent, failed to get token price")
			continue
		}
		now := time.Now()
		feedData := &common.FeedData{
			FeedID:    feed.ID,
			Value:     *price,
			Timestamp: &now,
		}
		log.Debug().Str("Player", "Uniswap").Any("feedData", feedData).Msg("price fetched")
		f.FeedDataBuffer <- feedData
		f.Mutex.Lock()
		f.LatestEntries[feed.ID] = feedData
		f.Mutex.Unlock()
	}
	return nil
}

func getTokenPrice(sqrtPrice *big.Int, definition *common.DexFeedDefinition) (*float64, error) {
	decimal0 := definition.Token0Decimals
	decimal1 := definition.Token1Decimals
	if sqrtPrice == nil || decimal0 == 0 || decimal1 == 0 {
		return nil, errorSentinel.ErrFetcherInvalidInput
	}

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

	multiplier := new(big.Float).SetFloat64(math.Pow(10, common.DECIMALS))
	datum.Mul(datum, multiplier)

	result, _ := datum.Float64()
	result = math.Round(result)

	return &result, nil
}
